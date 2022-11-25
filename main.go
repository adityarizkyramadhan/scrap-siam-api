package main

import (
	"errors"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly"
)

var (
	loginUrl  = "https://siam.ub.ac.id/index.php/"  //POST
	siamUrl   = "https://siam.ub.ac.id/"            //GET
	logoutUrl = "https://siam.ub.ac.id/logout.php/" //GET

	Version = "0.1.0"

	ErrorNotLoggedIn = errors.New("please login first")
	ErrorLoggedIn    = errors.New("already logged in")
)

type User struct {
	c       *colly.Collector
	Account struct {
		NIM          string
		Nama         string
		Jenjang      string
		Fakultas     string
		Jurusan      string
		ProgramStudi string
		Seleksi      string
		NomorUjian   string
	}
	LoginStatus bool
}

// constructor
func NewUser() User {
	return User{c: colly.NewCollector(), LoginStatus: false}
}

func (s *User) Login(us string, ps string) error {
	if s.LoginStatus {
		return ErrorLoggedIn
	}

	var errLogin error
	var doc *goquery.Document

	s.c.OnResponse(func(r *colly.Response) {
		doc, errLogin = goquery.NewDocumentFromReader(strings.NewReader(string(r.Body)))
		if errLogin != nil {
			errLogin = errors.New("couldn't read response body")
			return
		}
		temp := errors.New(strings.TrimSpace(doc.Find("small.error-code").Text()))
		if temp != nil {
			if len(temp.Error()) != 0 {
				errLogin = temp
				return
			}
		}
	})
	err := s.c.Post(loginUrl, map[string]string{
		"username": us,
		"password": ps,
		"login":    "Masuk",
	})

	if err != nil {
		if err.Error() != "Found" {
			return err
		}
	}
	if errLogin != nil {
		if len(errLogin.Error()) != 0 {
			return errLogin
		}
	}
	s.LoginStatus = true
	return nil
}

func (s *User) GetData() error {
	//scraping data mahasiswas
	result := make([]string, 9)
	s.c.OnHTML("div[class=\"bio-info\"]", func(h *colly.HTMLElement) {
		h.ForEach("div", func(i int, h *colly.HTMLElement) {
			each := strings.TrimSpace(h.Text)
			if each != "PDDIKTI KEMDIKBUDDetail" {
				result[i] = h.Text
			}
		})
	})
	s.c.OnHTML("div[class=\"photo-frame\"]", func(h *colly.HTMLElement) {
		h.ForEach("div", func(i int, h *colly.HTMLElement) {
			each := strings.TrimSpace(h.Text)
			if each != "PDDIKTI KEMDIKBUDDetail" {
				result[i] = h.Text
			}
		})
	})
	err := s.c.Visit(siamUrl)
	if err != nil {
		return err
	}

	s.Account.NIM = result[0]
	s.Account.Nama = result[1]
	// result2 = Jenjang/Fakultas--/--
	jenjangFakultas := strings.Split(result[2][16:], "/")
	s.Account.Jenjang = jenjangFakultas[0]
	s.Account.Fakultas = jenjangFakultas[1]
	s.Account.Jurusan = result[3][7:]
	s.Account.ProgramStudi = result[4][13:]
	s.Account.Seleksi = result[5][7:]
	s.Account.NomorUjian = result[6][11:]
	return nil
}

// make sure to defer this method after login, so the phpsessionid won't be misused
func (s *User) Logout() error {
	if !s.LoginStatus {
		return ErrorNotLoggedIn
	}
	s.c.Visit(logoutUrl)
	return nil
}

type mahasiswa struct {
	Nim      string `json:"nim"`
	Password string `json:"password"`
}

var cORS = func() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func main() {
	router := gin.Default()
	user := NewUser()
	router.Use(cORS())
	router.POST("/auth", func(ctx *gin.Context) {
		var input mahasiswa
		err := ctx.ShouldBindJSON(&input)
		if err != nil {
			log.Println(err.Error())
			return
		}
		err = user.Login(input.Nim, input.Password)
		if err != nil {
			log.Println(err.Error())
			return
		}
		err = user.GetData()
		if err != nil {
			log.Println(err.Error())
			return
		}
		err = user.Logout()
		if err != nil {
			log.Println(err.Error())
			return
		}
		ctx.JSON(200, gin.H{
			"success": true,
			"data":    user.Account,
		})
	})
	router.Run(":8081")
}
