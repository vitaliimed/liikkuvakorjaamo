package crm

// Drafts are private to the owner, including action responses.
func jobResponse(j Job, u User) map[string]any {
 out := map[string]any{"job": j}
 if u.Manager() { out["draft"] = Draft(j) }
 return out
}
func eventView(e Event, u User) (any, bool) {
 if u.Manager() { return e,true }
 text := ""
 switch e.Type {
 case "technical_note": text=e.Text
 case "internal_note": if e.Actor!=u.ID{return nil,false};text=e.Text
 case "created","assign","propose","client_confirm","confirm","reschedule","start","report","close","reopen","cancel","vehicle","photo":
 default: return nil,false
 }
 return map[string]any{"id":e.ID,"type":e.Type,"at":e.At,"revision":e.Revision,"text":text},true
}
func technicalVisit(j Job) map[string]any {
 return map[string]any{"id":j.ID,"created":j.Created,"status":j.Status,"description":j.Description,"model":j.Model,"mileage":j.Mileage,"photos":j.Photos,"report":map[string]any{"work":j.Report.Work,"remaining":j.Report.Remaining,"result":j.Report.Result}}
}
