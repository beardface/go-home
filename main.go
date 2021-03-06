package main

import (
	"encoding/json"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/stianeikeland/go-rpio"
	"gopkg.in/gomail.v1"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
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
	Door               string
	Open               bool
	RaspberryPiGpioPin rpio.Pin
	ImageURL           string
	ImageCount         int
	ImageDelayMs       int
}

var Home *HomeConfig
var Mail *MailConfig

func InitGpio() {
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for door := 0; door < len(Home.Doors); door++ {
		log.Printf("Initializing Door: %s on Pin: %d\n", Home.Doors[door].Door, Home.Doors[door].RaspberryPiGpioPin)
		Home.Doors[door].RaspberryPiGpioPin.Input()
		Home.Doors[door].RaspberryPiGpioPin.PullUp()
	}
}

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

	InitGpio()
}

func SendNotification(subject string, message string, imgFiles []string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", Mail.Username)
	msg.SetHeader("To", strings.Join(Mail.To, ","))
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", message)

	for i := 0; i < len(imgFiles); i++ {
		f, err := gomail.OpenFile(imgFiles[i])
		if err != nil {
			panic(err)
		}
		msg.Attach(f)
	}

	mailer := gomail.NewMailer(Mail.SmtpServer, Mail.Username, Mail.Password, Mail.SmtpPort)
	err := mailer.Send(msg)
	return err
}

func ReadDoorState(gpioPin rpio.Pin) bool {
	return gpioPin.Read() == 1
}

func downloadCameraImageToDisk(imgURL string, count int) (string, error) {
	fileName := fmt.Sprintf("snapshot%s.jpg", count)

	file, err := os.Create(fileName)

	if err != nil {
		return "", err
	}
	defer file.Close()

	check := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	resp, err := check.Get(imgURL)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)

	if err != nil {
		return "", err
	}

	log.Printf("%s with %v bytes downloaded", fileName, size)
	return fileName, nil
}

func StartUpdateStatusJob() {
	go func() {
		// Unmap gpio memory when done
		defer rpio.Close()
		for {
			//Check all Doors
			for door := 0; door < len(Home.Doors); door++ {
				doorState := ReadDoorState(Home.Doors[door].RaspberryPiGpioPin)
				if Home.Doors[door].Open != doorState {

					Home.Doors[door].Open = doorState

					state := "Closed"
					if Home.Doors[door].Open {
						state = "Opened"
					}
					at := time.Now().Format("Mon Jan _2 15:04:05 2006")

					doorEventText := fmt.Sprintf("%s has %s at %s",
						Home.Doors[door].Door,
						state,
						at)

					log.Println(doorEventText)

					imgFiles := []string{}
					if Home.Doors[door].ImageURL != "" {
						for i := 0; i < Home.Doors[door].ImageCount; i++ {
							imgFile, err := downloadCameraImageToDisk(Home.Doors[door].ImageURL, i)
							if err != nil {
								log.Println("Error calling downloadCameraImageToDisk:", err)
							} else {
								imgFiles = append(imgFiles, imgFile)
							}
							time.Sleep(time.Duration(Home.Doors[door].ImageDelayMs) * time.Millisecond)
						}
					}

					SendNotification(doorEventText,
						fmt.Sprintf(`<h3>Door Event Triggered</h3>
<ul>
	<li><b>Door:</b> %s</li>
	<li><b>State:</b> %s</li>
	<li><b>Time:</b> %s</li>
</ul>
`, Home.Doors[door].Door, state, at), imgFiles)
				}
			}
			time.Sleep(5000 * time.Millisecond)
		}
	}()
}

func main() {
	Init()

	ws := new(restful.WebService)
	ws.Path("/home").
		Consumes(restful.MIME_JSON, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_JSON)

	ws.Route(ws.GET("/status").To(homeStatus).
		Doc("get home status").
		Operation("homeStatus"))

	ws.Route(ws.GET("/test/email").To(testEmail).
		Doc("send test email").
		Operation("homeStatus"))

	restful.Add(ws)

	SendNotification("Home Monitor Has Started <eom>", "", []string{})
	log.Println("Starting Monitoring Process")
	StartUpdateStatusJob()

	log.Println("Starting Web Server on port 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Println("Failed to start http server:", err)
	}
}

func homeStatus(req *restful.Request, resp *restful.Response) {
	resp.WriteEntity(*Home)
}

func testEmail(req *restful.Request, resp *restful.Response) {
	err := SendNotification("Test Email From Home Monitor", "OK!", []string{})
	if err == nil {
		resp.WriteEntity("OK")
	} else {
		resp.WriteEntity(err)
	}
}
