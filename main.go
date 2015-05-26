package main

import (
	"encoding/json"
	"fmt"
	"github.com/emicklei/go-restful"
	"gopkg.in/gomail.v1"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type MailConfig struct {
	SmtpServer string
	SmtpPort   int
	Username   string
	Password   string
	To         []string
}

type HomeConfig struct {
	Doors []DoorStatus
}

type DoorStatus struct {
	Door string
	Open bool
}

var Home *HomeConfig
var Mail *MailConfig

func Init() {

	homeConfigFile, e := ioutil.ReadFile("./home.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}

	json.Unmarshal(homeConfigFile, &Home)

	emailConfigFile, e := ioutil.ReadFile("./email.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}

	json.Unmarshal(emailConfigFile, &Mail)
}

func SendNotification(subject string, message string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", Mail.Username)
	msg.SetHeader("To", strings.Join(Mail.To, ","))
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", message)

	mailer := gomail.NewMailer(Mail.SmtpServer, Mail.Username, Mail.Password, Mail.SmtpPort)
	err := mailer.Send(msg)
	return err
}

func main() {
	Init()

	ws := new(restful.WebService)
	ws.
		Path("/home").
		Consumes(restful.MIME_JSON, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_JSON) // you can specify this per route as well

	ws.Route(ws.GET("/status").To(homeStatus).
		Doc("get home status").
		Operation("homeStatus"))

	ws.Route(ws.GET("/test/email").To(testEmail).
		Doc("send test email").
		Operation("homeStatus"))

	restful.Add(ws)
	http.ListenAndServe(":8080", nil)
}

func homeStatus(req *restful.Request, resp *restful.Response) {
	resp.WriteEntity(*Home)
}

func testEmail(req *restful.Request, resp *restful.Response) {
	err := SendNotification("Test Email From Home Monitor", "OK!")
	if err == nil {
		resp.WriteEntity("OK")
	} else {
		resp.WriteEntity(err)
	}
}
