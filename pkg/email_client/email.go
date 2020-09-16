package email_client

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"message-center/pkg/configuration"
	"net/smtp"
	"strings"
)

type NotificationInterface interface {
	// 发送邮件
	SendEmail(nickname, subject, message string, receive Receive) error
}

type NotificationManagerImple struct {
	user     string
	password string
	host     string
	port     string
}

type (
	Receive struct {
		Ccer       []string // 抄送
		Recipients []string // 收件人
	}
	Body struct {
		ccer       []string
		recipients []string
		nickname   string
		subject    string
		message    string
	}

	loginAuth struct {
		username, password string
	}
)

func (p *NotificationManagerImple) Initialize(dcl configuration.ConfigurationLoader) {
	p.user = dcl.GetField("DevOps.Mgmt.API", "Email_SMTPSender")
	p.password = dcl.GetField("DevOps.Mgmt.API", "Email_SMTPPassword")
	p.host = dcl.GetField("DevOps.Mgmt.API", "Email_SMTPHost")
	p.port = dcl.GetField("DevOps.Mgmt.API", "Email_SMTPPort")
}

func (p *NotificationManagerImple) SendEmail(nickname, subject, message string, receive Receive) error {
	var toEmail []string
	toEmail = append(toEmail, receive.Ccer...)
	toEmail = append(toEmail, receive.Recipients...)

	bodyEmail := &Body{}
	bodyEmail.message = message
	bodyEmail.ccer = receive.Ccer
	bodyEmail.recipients = receive.Recipients
	bodyEmail.subject = subject
	bodyEmail.nickname = nickname

	var buffer strings.Builder
	buffer.WriteString(p.host)
	buffer.WriteString(":")
	buffer.WriteString(p.port)

	Host := buffer.String()

	var authorized smtp.Auth
	authorized = &loginAuth{p.user, p.password}

	var cushion bytes.Buffer
	cc := strings.Join(bodyEmail.ccer, ",")
	re := strings.Join(bodyEmail.recipients, ",")
	cushion.WriteString("To:")
	cushion.WriteString(re)
	cushion.WriteString("\r\n")
	cushion.WriteString("Cc:")
	cushion.WriteString(cc)
	cushion.WriteString("\r\nFrom:")
	cushion.WriteString(nickname)
	cushion.WriteString("<")
	cushion.WriteString(p.user)
	cushion.WriteString(">\r\nSubject: ")
	cushion.WriteString(subject)
	cushion.WriteString("\r\n")
	cushion.WriteString("Content-Type: text/html;charset=UTF-8")
	cushion.WriteString("\r\n\r\n")
	cushion.WriteString(bodyEmail.message)
	msg := cushion.Bytes()

	err := sendMailUsingTLS(
		Host,
		authorized,
		p.user,
		toEmail,
		msg,
	)

	if err != nil {
		log.Println("发送失败:", err)
		return err
	}
	return nil
}

func sendMailUsingTLS(addr string, auth smtp.Auth, from string,
	to []string, msg []byte) (err error) {
	log.Println("start SendMailUsingTLS... ")

	c, err := dial(addr)

	log.Println("c:", c)
	if err != nil {
		fmt.Println("Create smtp client error:", err)
		return err
	}

	defer c.Close()

	if auth != nil {
		if ok, param := c.Extension("AUTH"); ok {
			fmt.Println("ok:", ok)
			fmt.Println("param:", param)
			if err = c.Auth(auth); err != nil {
				fmt.Println("Error during AUTH:", err)
				return err
			}
		}
	}
	if err = c.Mail(from); err != nil {
		fmt.Println(err)
		return err
	}

	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			fmt.Println(err)
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		fmt.Println(err)
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = w.Close()
	if err != nil {
		fmt.Println(err)
		return err
	}
	return c.Quit()
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New(" Unknown fromServer")
		}
	}
	return nil, nil
}

func dial(addr string) (*smtp.Client, error) {
	conn, err := smtp.Dial(addr)
	if err != nil {
		fmt.Println("Dialing Error:", err)
		return nil, err
	}

	return conn, nil
}
