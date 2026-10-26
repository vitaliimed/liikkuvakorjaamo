package main
import("bytes";"crypto/tls";"encoding/base64";"fmt";"io";"mime";"mime/multipart";"mime/quotedprintable";"net";"net/http";"net/mail";"net/smtp";"net/textproto";"path/filepath";"strings";"time")
const maxEmailAttachmentBytes=15<<20
type EmailAttachment struct{Name,ContentType string;Data []byte}
func readEmailAttachments(r *http.Request)([]EmailAttachment,error){
 if r.MultipartForm==nil{return nil,nil};files:=r.MultipartForm.File["custom_email_attachments"];if len(files)>10{return nil,fmt.Errorf("Допускается до 10 вложений")}
 var result []EmailAttachment;total:=0
 for _,header:=range files{f,err:=header.Open();if err!=nil{return nil,err};b,err:=io.ReadAll(io.LimitReader(f,int64(maxEmailAttachmentBytes-total+1)));f.Close();if err!=nil{return nil,err};total+=len(b);if total>maxEmailAttachmentBytes{return nil,fmt.Errorf("Суммарный размер вложений превышает 15 МБ")};name:=filepath.Base(strings.ReplaceAll(header.Filename,"\\","/"));if len(name)>200||strings.ContainsAny(name,"\r\n\x00"){return nil,fmt.Errorf("Некорректное имя файла")};ct:=http.DetectContentType(b);result=append(result,EmailAttachment{name,ct,b})};return result,nil
}
func buildSMTPMessage(cfg Config,to,subject,textBody,htmlBody string,attachments ...EmailAttachment)([]byte,error){
 for _,v:=range []string{cfg.MailFrom,cfg.MailReplyTo,to,subject}{if strings.ContainsAny(v,"\r\n"){return nil,fmt.Errorf("Недопустимый заголовок письма")}}
 if _,err:=mail.ParseAddress(to);err!=nil{return nil,err};if _,err:=mail.ParseAddress(cfg.MailFrom);err!=nil{return nil,err}
 var alt bytes.Buffer;aw:=multipart.NewWriter(&alt)
 for _,part:=range []struct{kind,body string}{{"text/plain",textBody},{"text/html",htmlBody}}{h:=textproto.MIMEHeader{};h.Set("Content-Type",part.kind+`; charset="UTF-8"`);h.Set("Content-Transfer-Encoding","quoted-printable");w,err:=aw.CreatePart(h);if err!=nil{return nil,err};q:=quotedprintable.NewWriter(w);if _,err=q.Write([]byte(part.body));err!=nil{return nil,err};if err=q.Close();err!=nil{return nil,err}}
 if err:=aw.Close();err!=nil{return nil,err};kind:=mime.FormatMediaType("multipart/alternative",map[string]string{"boundary":aw.Boundary()});payload:=alt.Bytes()
 if len(attachments)>0{var mixed bytes.Buffer;mw:=multipart.NewWriter(&mixed);h:=textproto.MIMEHeader{};h.Set("Content-Type",kind);w,err:=mw.CreatePart(h);if err!=nil{return nil,err};w.Write(payload);total:=0
  for _,a:=range attachments{total+=len(a.Data);if total>maxEmailAttachmentBytes||len(attachments)>10{return nil,fmt.Errorf("Слишком много вложений")};h=textproto.MIMEHeader{};h.Set("Content-Type","application/octet-stream");h.Set("Content-Disposition",mime.FormatMediaType("attachment",map[string]string{"filename":a.Name}));h.Set("Content-Transfer-Encoding","base64");w,err=mw.CreatePart(h);if err!=nil{return nil,err};encoded:=base64.StdEncoding.EncodeToString(a.Data);for len(encoded)>76{io.WriteString(w,encoded[:76]+"\r\n");encoded=encoded[76:]};io.WriteString(w,encoded+"\r\n")};if err=mw.Close();err!=nil{return nil,err};payload=mixed.Bytes();kind=mime.FormatMediaType("multipart/mixed",map[string]string{"boundary":mw.Boundary()})
 }
 var result bytes.Buffer;fmt.Fprintf(&result,"From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s@liikkuvakorjaamo.fi>\r\nMIME-Version: 1.0\r\nContent-Type: %s\r\n",cfg.MailFrom,to,mime.QEncoding.Encode("UTF-8",subject),time.Now().Format(time.RFC1123Z),liikkuvaSubmissionID(),kind)
 if cfg.MailReplyTo!=""{fmt.Fprintf(&result,"Reply-To: %s\r\n",cfg.MailReplyTo)};result.WriteString("\r\n");result.Write(payload);return result.Bytes(),nil
}
func liikkuvaSendSMTP(cfg Config,to,subject,textBody,htmlBody string,attachments ...EmailAttachment)error{
 data,err:=buildSMTPMessage(cfg,to,subject,textBody,htmlBody,attachments...);if err!=nil{return err}
 host:=strings.TrimSpace(cfg.SMTPHost);address:=net.JoinHostPort(host,strings.TrimSpace(cfg.SMTPPort));dialer:=&net.Dialer{Timeout:15*time.Second};var conn net.Conn
 tc:=&tls.Config{ServerName:host,MinVersion:tls.VersionTLS12}
 if cfg.SMTPPort=="465"{conn,err=tls.DialWithDialer(dialer,"tcp",address,tc)}else{conn,err=dialer.Dial("tcp",address)};if err!=nil{return err};defer conn.Close();conn.SetDeadline(time.Now().Add(45*time.Second))
 c,err:=smtp.NewClient(conn,host);if err!=nil{return err};defer c.Close();if cfg.SMTPPort!="465"{if ok,_:=c.Extension("STARTTLS");!ok{return fmt.Errorf("SMTP требует TLS; сервер не предлагает STARTTLS")};if err=c.StartTLS(tc);err!=nil{return err}}
 if err=c.Auth(smtp.PlainAuth("",cfg.SMTPUsername,cfg.SMTPPassword,host));err!=nil{return err};from,err:=mail.ParseAddress(cfg.MailFrom);if err!=nil{return err};if err=c.Mail(from.Address);err!=nil{return err}
 seen:=map[string]bool{};for _,recipient:=range []string{to,cfg.MailCopyTo}{if recipient==""||seen[strings.ToLower(recipient)]{continue};seen[strings.ToLower(recipient)]=true;a,e:=mail.ParseAddress(recipient);if e!=nil{return e};if e=c.Rcpt(a.Address);e!=nil{return e}}
 w,err:=c.Data();if err!=nil{return err};if _,err=w.Write(data);err!=nil{return err};if err=w.Close();err!=nil{return err};return c.Quit()
}
