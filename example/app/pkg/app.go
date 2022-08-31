package build

import (
	"database/sql"
	"fmt"
	"math/rand"

	_ "github.com/mattn/go-sqlite3"
)

// FIXME(paulsmith): package global db conn
var DB *sql.DB

// FIXME(paulsmith): relying on init() is not great from an app lifecycle POV
func init() {
	dbPath := "./mypushupapp.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(fmt.Errorf("opening SQLite db %s: %w", dbPath, err))
	}
	if err := initDb(db); err != nil {
		panic(fmt.Errorf("initializing SQLite db: %w", err))
	}
	DB = db
}

var createTable = `
CREATE TABLE IF NOT EXISTS [albums] (
	[id] INTEGER PRIMARY KEY,
	[artist] TEXT,
	[title] TEXT,
	[released] INTEGER,
	[length] INTEGER
);
`

func initDb(db *sql.DB) error {
	if _, err := db.Exec(createTable); err != nil {
		return fmt.Errorf("creating albums table: %w", err)
	}
	return nil
}

type album struct {
	id       int
	artist   string
	title    string
	released int
	length   int
}

var insertAlbumRow = `
INSERT INTO [albums] ([artist], [title], [released], [length])
VALUES (?, ?, ?, ?)
RETURNING [id]
`

func addAlbum(db *sql.DB, a *album) error {
	args := []any{
		a.artist,
		a.title,
		a.released,
		a.length,
	}
	if err := db.QueryRow(insertAlbumRow, args...).Scan(&a.id); err != nil {
		return fmt.Errorf("inserting album: %w", err)
	}
	return nil
}

var selectAlbumById = `
SELECT [artist], [title], [released], [length]
FROM [albums]
WHERE [id] = ?
`

func getAlbumById(db *sql.DB, id int) (*album, error) {
	a := album{id: id}
	dest := []any{
		&a.artist,
		&a.title,
		&a.released,
		&a.length,
	}
	if err := db.QueryRow(selectAlbumById, id).Scan(dest...); err != nil {
		return nil, fmt.Errorf("getting album by ID: %w", err)
	}
	return &a, nil
}

var selectAlbums = `
SELECT [id], [artist], [title], [released], [length]
FROM [albums]
ORDER BY id
`

func getAlbums(db *sql.DB, limit, offset int) ([]*album, error) {
	query := selectAlbums
	var args []any
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("getting albums: %v", err)
	}
	defer rows.Close()
	var albums []*album
	for rows.Next() {
		var a album
		dest := []any{
			&a.id,
			&a.artist,
			&a.title,
			&a.released,
			&a.length,
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("getting albums, scanning row: %w", err)
		}
		albums = append(albums, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("getting albums, scan: %w", err)
	}
	return albums, nil
}

var updateAlbum = `
UPDATE [albums]
SET
	[artist] = ?,
	[title] = ?,
	[released] = ?,
	[length] = ?
WHERE [id] = ?
`

func editAlbum(db *sql.DB, id int, a *album) error {
	args := []any{
		&a.artist,
		&a.title,
		&a.released,
		&a.length,
		id,
	}
	if _, err := db.Exec(updateAlbum, args...); err != nil {
		return fmt.Errorf("updating album: %v", err)
	}
	return nil
}

var deleteAlbum_ = `DELETE FROM [albums] WHERE [id] = ?`

func deleteAlbum(db *sql.DB, id int) error {
	if _, err := db.Exec(deleteAlbum_, id); err != nil {
		return fmt.Errorf("deleting album: %v", err)
	}
	return nil
}

var fakeNames = []string{
	"Patty O’Furniture",
	"Paddy O’Furniture",
	"Olive Yew",
	"Aida Bugg",
	"Maureen Biologist",
	"Teri Dactyl",
	"Peg Legge",
	"Allie Grater",
	"Liz Erd",
	"A. Mused",
	"Constance Noring",
	"Lois Di Nominator",
	"Minnie Van Ryder",
	"Lynn O’Leeum",
	"P. Ann O’Recital",
	"Ray O’Sun",
	"Lee A. Sun",
	"Ray Sin",
	"Isabelle Ringing",
	"Eileen Sideways",
	"Rita Book",
	"Paige Turner",
	"Rhoda Report",
	"Augusta Wind",
	"Chris Anthemum",
	"Anne Teak",
	"U.R. Nice",
	"Anita Bath",
	"Harriet Upp",
	"I.M. Tired",
	"I. Missy Ewe",
	"Ivana B. Withew",
	"Anita Letterback",
	"Hope Furaletter",
	"B. Homesoon",
	"Bea Mine",
	"Bess Twishes",
	"C. Yasoon",
	"Audie Yose",
	"Dee End",
	"Amanda Hug",
	"Ben Dover",
	"Eileen Dover",
	"Willie Makit",
	"Willie Findit",
	"Skye Blue",
	"Staum Clowd",
	"Addie Minstra",
	"Anne Ortha",
	"Dave Allippa",
	"Dee Zynah",
	"Hugh Mannerizorsa",
	"Loco Lyzayta",
	"Manny Jah",
	"Mark Ateer",
	"Reeve Ewer",
	"Tex Ryta",
	"Theresa Green",
	"Barry Kade",
	"Stan Dupp",
	"Neil Down",
	"Con Trariweis",
	"Don Messwidme",
	"Al Annon",
	"Anna Domino",
	"Clyde Stale",
	"Anna Logwatch",
	"Anna Littlical",
	"Norma Leigh Absent",
	"Sly Meebuggah",
	"Saul Goodmate",
	"Faye Clether",
	"Sarah Moanees",
	"Ty Ayelloribbin",
	"Hugo First",
	"Percy Vere",
	"Jack Aranda",
	"Olive Tree",
	"Fran G. Pani",
	"John Quil",
	"Ev R. Lasting",
	"Anne Thurium",
	"Cherry Blossom",
	"Glad I. Oli",
	"Ginger Plant",
	"Del Phineum",
	"Rose Bush",
	"Perry Scope",
	"Frank N. Stein",
	"Roy L. Commishun",
	"Pat Thettick",
	"Percy Kewshun",
	"Rod Knee",
	"Hank R. Cheef",
	"Bridget Theriveaquai",
	"Pat N. Toffis",
	"Karen Onnabit",
	"Col Fays",
	"Fay Daway",
	"Joe V. Awl",
	"Wes Yabinlatelee",
	"Colin Sik",
	"Greg Arias",
	"Toi Story",
	"Gene Eva Convenshun",
	"Jen Tile",
	"Simon Sais",
	"Peter Owt",
	"Hugh N. Cry",
	"Lee Nonmi",
	"Lynne Gwafranca",
	"Art Decco",
	"Lynne Gwistic",
	"Polly Ester Undawair",
	"Oscar Nommanee",
	"Laura Biding",
	"Laura Norda",
	"Des Ignayshun",
	"Mike Rowe-Soft",
	"Anne T. Kwayted",
	"Wayde N. Thabalanz",
	"Dee Mandingboss",
	"Sly Meedentalfloss",
	"Stanley Knife",
	"Wynn Dozeaplikayshun",
	"Mal Ajusted",
	"Penny Black",
	"Mal Nurrisht",
	"Polly Pipe",
	"Polly Wannakrakouer",
	"Con Staninterupshuns",
	"Fran Tick",
	"Santi Argo",
	"Carmen Goh",
	"Carmen Sayid",
	"Norma Stitts",
	"Ester La Vista",
	"Manuel Labor",
	"Ivan Itchinos",
	"Ivan Notheridiya",
	"Mustafa Leek",
	"Emma Grate",
	"Annie Versaree",
	"Tim Midsaylesman",
	"Mary Krismass",
	"Tim “Buck” Too",
	"Lana Lynne Creem",
	"Wiley Waites",
	"Ty R. Leeva",
	"Ed U. Cayshun",
	"Anne T. Dote",
	"Claude Strophobia",
	"Anne Gloindian",
	"Dulcie Veeta",
	"Abby Normal",
}

var alphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func uid() string {
	b := make([]rune, 16)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}
