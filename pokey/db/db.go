package db

import (
	"fmt"
	"os"

	"github.com/PuerkitoBio/goquery"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
)

type Tag struct {
	Id     int64   `json:"-"`
	Name   string  `sql:"size:255;not null;unique" json:"name"`
	Comics []Comic `gorm:"many2many:pokey_comic_tags;" json:"-"`
}

func (t Tag) TableName() string {
	return "pokey_tag"
}

type Image struct {
	Id      int64  `json:"-"`
	ComicId int64  `json:"-"`
	Order   int16  `sql:"not null" json:"-"`
	Url     string `sql:"size:255;not null" json:"url"`
}

func (i Image) TableName() string {
	return "pokey_image"
}

type Comic struct {
	Id     int64   `json:"-"`
	Index  uint32  `sql:"not null;unique" json:"index"`
	Title  string  `sql:"size:255;not null" json:"title"`
	Url    string  `sql:"size:255;not null;unique" json:"url"`
	Image  string  `sql:"size:255;not null;unique" json:"image"`
	Images []Image `json:"images,omitempty"`
	Tags   []Tag   `gorm:"many2many:pokey_comic_tags;" json:"tags,omitempty"`
}

func (c Comic) TableName() string {
	return "pokey_comic"
}

type Database struct {
	db gorm.DB
}

func Open() (Database, error) {
	connection, err := pq.ParseURL(os.Getenv("DATABASE_URL"))
	if err != nil {
		return Database{}, err
	}

	sslmode := os.Getenv("PGSSL")
	if sslmode == "" {
		sslmode = "disable"
	}
	connection += " sslmode=" + sslmode

	db, err := gorm.Open("postgres", connection)
	if err != nil {
		return Database{}, err
	}

	return Database{db: db}, nil
}

func (d *Database) addForeignKey(tableName string,
	columnName string,
	foreignTableName string,
	foreignColumnName string,
	constraintName string) {
	d.db.Exec(fmt.Sprintf(
		"ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s) ON UPDATE CASCADE ON DELETE CASCADE",
		tableName, constraintName, columnName, foreignTableName, foreignColumnName))
}

func (d *Database) Init() {
	if !d.db.HasTable(&Comic{}) {
		d.db.CreateTable(&Comic{})
		d.db.Model(&Comic{}).AddUniqueIndex("pokey_comic_index_idx", "index")
		d.db.Model(&Comic{}).AddIndex("pokey_comic_title_idx", "title")
	}

	if !d.db.HasTable(&Image{}) {
		d.db.CreateTable(Image{})
		d.db.Model(&Image{}).AddIndex("pokey_image_url_idx", "url")
		d.addForeignKey(Image{}.TableName(), "comic_id", Comic{}.TableName(), "id", "pokey_image_comic_fk")
	}

	if !d.db.HasTable(&Tag{}) {
		d.db.CreateTable(&Tag{})
		d.db.Model(&Tag{}).AddUniqueIndex("pokey_tag_name_idx", "name")
		d.addForeignKey("pokey_comic_tags", "comic_id", Comic{}.TableName(), "id", "pokey_comic_tags_comic_fk")
		d.addForeignKey("pokey_comic_tags", "tag_id", Tag{}.TableName(), "id", "pokey_comic_tags_tag_fk")
	}
}

const POKEY_ARCHIVE_URL = "http://www.yellow5.com/pokey/archive/"

func (d *Database) populateComic(index uint32, title string, url string) {
	imageUrl := fmt.Sprintf("%spokey%d.gif", POKEY_ARCHIVE_URL, index)
	comic := Comic{Index: index, Title: title, Url: url, Image: imageUrl}

	doc, err := goquery.NewDocument(url)
	if err != nil {
		panic(err)
	}

	doc.Find("img[src#=pokey(\\d+_\\d+)?\\.gif]").Each(func(i int, s *goquery.Selection) {
		order := int16(i) + 1
		src, _ := s.Attr("src")
		comic.Images = append(comic.Images, Image{Order: order, Url: POKEY_ARCHIVE_URL + src})
	})

	d.db.Create(&comic)
}

func (d *Database) Populate() {
	doc, err := goquery.NewDocument(POKEY_ARCHIVE_URL)
	if err != nil {
		panic(err)
	}

	doc.Find("a[href#=index\\d+\\.html]").Each(func(i int, s *goquery.Selection) {
		index := uint32(i) + 1
		title := s.Find("i").Text()
		href, _ := s.Attr("href")
		d.populateComic(index, title, POKEY_ARCHIVE_URL+href)
	})
}

func (d *Database) Clear() {
	d.db.DropTableIfExists(&Image{})
	d.db.Exec("DROP TABLE IF EXISTS pokey_comic_tags")
	d.db.DropTableIfExists(&Tag{})
	d.db.DropTableIfExists(&Comic{})
}

func (d *Database) LoadAllComics(comics *[]Comic, loadImages bool, loadTags bool) {
	d.db.Order("index ASC").Find(comics)

	for i := range *comics {
		if loadImages {
			d.LoadImages(&(*comics)[i])
		}
		if loadTags {
			d.LoadTags(&(*comics)[i])
		}
	}
}

func (d *Database) LoadImages(comic *Comic) {
	d.db.Model(comic).Related(&comic.Images).Order("order ASC")
}

func (d *Database) LoadTags(comic *Comic) {
	d.db.Model(comic).Association("Tags").Find(&comic.Tags)
}
