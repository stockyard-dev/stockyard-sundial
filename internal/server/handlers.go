package server
import("encoding/json";"net/http";"strconv";"github.com/stockyard-dev/stockyard-sundial/internal/store")
func(s *Server)handleListProjects(w http.ResponseWriter,r *http.Request){list,_:=s.db.ListProjects();if list==nil{list=[]store.Project{}};writeJSON(w,200,list)}
func(s *Server)handleCreateProject(w http.ResponseWriter,r *http.Request){var p store.Project;json.NewDecoder(r.Body).Decode(&p);if p.Name==""{writeError(w,400,"name required");return};s.db.CreateProject(&p);writeJSON(w,201,p)}
func(s *Server)handleStart(w http.ResponseWriter,r *http.Request){id,_:=strconv.ParseInt(r.PathValue("id"),10,64);var req struct{Description string `json:"description"`};json.NewDecoder(r.Body).Decode(&req);e,_:=s.db.Start(id,req.Description);writeJSON(w,201,e)}
func(s *Server)handleStop(w http.ResponseWriter,r *http.Request){id,_:=strconv.ParseInt(r.PathValue("id"),10,64);s.db.Stop(id);writeJSON(w,200,map[string]string{"status":"stopped"})}
func(s *Server)handleListEntries(w http.ResponseWriter,r *http.Request){id,_:=strconv.ParseInt(r.PathValue("id"),10,64);list,_:=s.db.ListEntries(id);if list==nil{list=[]store.Entry{}};writeJSON(w,200,list)}
func(s *Server)handleDelete(w http.ResponseWriter,r *http.Request){id,_:=strconv.ParseInt(r.PathValue("id"),10,64);s.db.Delete(id);writeJSON(w,200,map[string]string{"status":"deleted"})}
func(s *Server)handleOverview(w http.ResponseWriter,r *http.Request){m,_:=s.db.Stats();writeJSON(w,200,m)}
