package crm

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "net/url"
    "strings"
    "testing"
    "time"

    "golang.org/x/crypto/bcrypt"
)

func TestCRMPassRulesAndRecoveryLink(t *testing.T) {
    cases := []struct{ password string; want bool }{
        {"",false},{" ",false},{"\t\n",false},{"x",true},
        {strings.Repeat("a",72),true},{strings.Repeat("a",73),false},
        {"Correct Horse Battery Staple!",true},
    }
    for _, c := range cases {
        if got:=validCRMPassword(c.password);got!=c.want {
            t.Errorf("password length %d: got %v want %v",len(c.password),got,c.want)
        }
    }
    token:=strings.Repeat("a",64)
    link,err:=passwordResetLink("https://liikkuvakorjaamo.fi",token)
    if err!=nil || link!="https://liikkuvakorjaamo.fi/crm/#reset="+token {t.Fatalf("%q %v",link,err)}
    for _,base:=range []string{"http://liikkuvakorjaamo.fi","https://liikkuvakorjaamo.fi/redirect","https://liikkuvakorjaamo.fi?next=evil","https://u@liikkuvakorjaamo.fi"} {
        if _,err:=passwordResetLink(base,token);err==nil {t.Errorf("accepted unsafe URL %q",base)}
    }
    if _,err:=passwordResetLink("https://liikkuvakorjaamo.fi","short");err==nil {t.Fatal("accepted short token")}
}

func TestPasswordResetHTMLHasSafeClickableLink(t *testing.T) {
    e:=Email{From:"service@liikkuvakorjaamo.fi",To:"owner@example.test",
        Subject:"Восстановление", Body:"Use the link", Language:"ru",
        ResetURL:"https://liikkuvakorjaamo.fi/crm/#reset="+strings.Repeat("b",64)}
    html:=brandedEmailHTML(e)
    if !strings.Contains(html,"href=\"https://liikkuvakorjaamo.fi/crm/#reset=") ||
       !strings.Contains(html,"Установить новый пароль") || !strings.Contains(html,"#23372b") {
        t.Fatal("missing clickable branded CRM reset link")
    }
    e.ResetURL="javascript:alert(1)"
    if strings.Contains(brandedEmailHTML(e),"href=\"javascript:") {t.Fatal("unsafe reset link accepted")}
}

func TestEmulatorCRMRegisteredOwnerPasswordReset(t *testing.T){
    t.Setenv("MAIL_FROM","service@liikkuvakorjaamo.fi")
    t.Setenv("SMTP_HOST","smtp.example.test")
    t.Setenv("SMTP_PORT","587")
    s:=emulator(t)
    ctx:=context.Background()
    email:="owner@example.test"
    id:=digest(email)
    oldPassword:="original-long-password"
    hash,err:=bcrypt.GenerateFromPassword([]byte(oldPassword),12)
    if err!=nil {t.Fatal(err)}
    user:=User{ID:id,Name:"Owner",Email:email,Role:"owner",Hash:string(hash),Active:true}
    if _,err:=s.Collection("users").Doc(id).Create(ctx,user);err!=nil {t.Fatal(err)}
    tokenSession:=randomID()
    if _,err:=s.Collection("sessions").Doc(digest(tokenSession)).Create(ctx,session{
        User:id,Auth:digest(string(hash)),Expires:time.Now().Add(time.Hour),
    });err!=nil {t.Fatal(err)}
    authReq:=httptest.NewRequest("GET","/crm/api/me",nil)
    authReq.AddCookie(&http.Cookie{Name:"crm_session",Value:tokenSession})
    if _,err:=s.auth(authReq);err!=nil{t.Fatal("old session must be active",err)}
    var mailed []Email
    s.resetMailer=func(e Email)(string,error){mailed=append(mailed,e);return "accepted",nil}
    api:=func(path string,body any)*httptest.ResponseRecorder{
        raw,_:=json.Marshal(body)
        req:=httptest.NewRequest("POST","/crm/api/password/"+path,strings.NewReader(string(raw)))
        req.Header.Set("Content-Type","application/json")
        req.Header.Set("X-CRM-Request","1")
        w:=httptest.NewRecorder()
        s.ServeHTTP(w,req)
        return w
    }
    unknown:=api("forgot",map[string]string{"email":"unknown@example.test"})
    known:=api("forgot",map[string]string{"email":email})
    if known.Code!=200||unknown.Code!=200||known.Body.String()!=unknown.Body.String(){t.Fatalf("account enumeration leak: %s / %s",known.Body.String(),unknown.Body.String())}
    if len(mailed)!=1||mailed[0].To!=email {t.Fatalf("registered reset mail missing: %+v",mailed)}
    link,err:=url.Parse(mailed[0].ResetURL)
    if err!=nil || !strings.Contains(link.Fragment,"reset="){t.Fatal("reset link has no URL fragment",err)}
    token:=strings.TrimPrefix(link.Fragment,"reset=")
    if len(token)!=64 {t.Fatal("reset token is not 256-bit hex")}
    if w:=api("verify",map[string]string{"token":token});w.Code!=200||!strings.Contains(w.Body.String(),email){t.Fatal("registered email not prefilled",w.Code,w.Body.String())}
    if w:=api("reset",map[string]string{"token":token,"password":"z"});w.Code!=200{t.Fatal("any nonempty password should work",w.Code,w.Body.String())}
    if _,err:=s.auth(authReq);err==nil {t.Fatal("old session remained valid after password reset")}
    if w:=api("verify",map[string]string{"token":token});w.Code!=400{t.Fatal("one-time reset token reused",w.Code)}
    if w:=api("reset",map[string]string{"token":token,"password":"other"});w.Code!=400{t.Fatal("same reset token accepted twice",w.Code)}
    doc,err:=s.Collection("users").Doc(id).Get(ctx)
    if err!=nil{t.Fatal(err)}
    var now User
    if err=doc.DataTo(&now);err!=nil {t.Fatal(err)}
    if bcrypt.CompareHashAndPassword([]byte(now.Hash),[]byte("z"))!=nil {t.Fatal("new password not saved")}
    if bcrypt.CompareHashAndPassword([]byte(now.Hash),[]byte(oldPassword))==nil {t.Fatal("old password still valid")}
    if w:=api("forgot",map[string]string{"email":email});w.Code!=200{t.Fatal("request rate limit rejected reset",w.Code)}
    if len(mailed)!=2 {t.Fatal("second recovery email not sent",len(mailed))}
}

func TestEmulatorCRMPasswordResetExpirationsAndUserScope(t *testing.T){
    s:=emulator(t)
    ctx:=context.Background()
    email:="mechanic@example.test"
    id:=digest(email)
    hashed,_:=bcrypt.GenerateFromPassword([]byte("mechanic"),12)
    user:=User{ID:id,Email:email,Role:"mechanic",Active:true,Hash:string(hashed)}
    if _,err:=s.Collection("users").Doc(id).Create(ctx,user);err!=nil{t.Fatal(err)}
    token:=randomID()
    record:=passwordResetRecord{User:id,Auth:digest(user.Hash),Expires:time.Now().Add(-time.Minute)}
    if _,err:=s.Collection("password_resets").Doc(digest(token)).Create(ctx,record);err!=nil{t.Fatal(err)}
    if _,_,err:=s.validPasswordReset(ctx,token);err==nil {t.Fatal("expired token accepted")}
    record.Expires=time.Now().Add(time.Minute)
    _,err:=s.Collection("password_resets").Doc(digest(token)).Set(ctx,record)
    if err!=nil{t.Fatal(err)}
    if _,u,err:=s.validPasswordReset(ctx,token);err!=nil||u.Email!=email{t.Fatal("mechanic recovery failed",err)}
    user.Active=false
    if _,err:=s.Collection("users").Doc(id).Set(ctx,user);err!=nil{t.Fatal(err)}
    if _,_,err:=s.validPasswordReset(ctx,token);err==nil{t.Fatal("inactive account reset accepted")}
}
