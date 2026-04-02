package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type TimeEntry struct {
	ID string `json:"id"`
	Description string `json:"description"`
	Project string `json:"project"`
	Task string `json:"task"`
	Duration int `json:"duration"`
	StartTime string `json:"start_time"`
	EndTime string `json:"end_time"`
	Billable int `json:"billable"`
	Tags string `json:"tags"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"sundial.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS time_entries(id TEXT PRIMARY KEY,description TEXT NOT NULL,project TEXT DEFAULT '',task TEXT DEFAULT '',duration INTEGER DEFAULT 0,start_time TEXT DEFAULT '',end_time TEXT DEFAULT '',billable INTEGER DEFAULT 1,tags TEXT DEFAULT '',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *TimeEntry)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO time_entries(id,description,project,task,duration,start_time,end_time,billable,tags,created_at)VALUES(?,?,?,?,?,?,?,?,?,?)`,e.ID,e.Description,e.Project,e.Task,e.Duration,e.StartTime,e.EndTime,e.Billable,e.Tags,e.CreatedAt);return err}
func(d *DB)Get(id string)*TimeEntry{var e TimeEntry;if d.db.QueryRow(`SELECT id,description,project,task,duration,start_time,end_time,billable,tags,created_at FROM time_entries WHERE id=?`,id).Scan(&e.ID,&e.Description,&e.Project,&e.Task,&e.Duration,&e.StartTime,&e.EndTime,&e.Billable,&e.Tags,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]TimeEntry{rows,_:=d.db.Query(`SELECT id,description,project,task,duration,start_time,end_time,billable,tags,created_at FROM time_entries ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []TimeEntry;for rows.Next(){var e TimeEntry;rows.Scan(&e.ID,&e.Description,&e.Project,&e.Task,&e.Duration,&e.StartTime,&e.EndTime,&e.Billable,&e.Tags,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Update(e *TimeEntry)error{_,err:=d.db.Exec(`UPDATE time_entries SET description=?,project=?,task=?,duration=?,start_time=?,end_time=?,billable=?,tags=? WHERE id=?`,e.Description,e.Project,e.Task,e.Duration,e.StartTime,e.EndTime,e.Billable,e.Tags,e.ID);return err}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM time_entries WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM time_entries`).Scan(&n);return n}

func(d *DB)Search(q string, filters map[string]string)[]TimeEntry{
    where:="1=1"
    args:=[]any{}
    if q!=""{
        where+=" AND (description LIKE ?)"
        args=append(args,"%"+q+"%");
    }
    rows,_:=d.db.Query(`SELECT id,description,project,task,duration,start_time,end_time,billable,tags,created_at FROM time_entries WHERE `+where+` ORDER BY created_at DESC`,args...)
    if rows==nil{return nil};defer rows.Close()
    var o []TimeEntry;for rows.Next(){var e TimeEntry;rows.Scan(&e.ID,&e.Description,&e.Project,&e.Task,&e.Duration,&e.StartTime,&e.EndTime,&e.Billable,&e.Tags,&e.CreatedAt);o=append(o,e)};return o
}

func(d *DB)Stats()map[string]any{
    m:=map[string]any{"total":d.Count()}
    return m
}
