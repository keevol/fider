package mailgun_test

import (
	"io/ioutil"
	"net/url"
	"testing"

	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/worker"

	"github.com/getfider/fider/app/models"
	"github.com/getfider/fider/app/pkg/email"
	"github.com/getfider/fider/app/pkg/mock"

	"github.com/getfider/fider/app/pkg/email/mailgun"
	"github.com/getfider/fider/app/pkg/log/noop"

	. "github.com/getfider/fider/app/pkg/assert"
)

var logger = noop.NewLogger()
var client = mock.NewHTTPClient()
var sender = mailgun.NewSender(logger, client, "mydomain.com", "mys3cr3tk3y")
var tenant = &models.Tenant{
	Subdomain: "got",
}
var ctx = worker.NewContext("ID-1", worker.Task{Name: "TaskName"}, nil, logger)

func init() {
	ctx.SetTenant(tenant)
}

func TestSend_Success(t *testing.T) {
	RegisterT(t)
	env.Config.HostMode = "multi"
	client.Reset()

	to := email.Recipient{
		Name:    "Jon Sow",
		Address: "jon.snow@got.com",
	}
	sender.Send(ctx, "echo_test", email.Params{
		"name": "Hello",
	}, "Fider Test", to)

	Expect(client.Requests).HasLen(1)
	Expect(client.Requests[0].URL.String()).Equals("https://api.mailgun.net/v3/mydomain.com/messages")
	Expect(client.Requests[0].Header.Get("Authorization")).Equals("Basic YXBpOm15czNjcjN0azN5")
	Expect(client.Requests[0].Header.Get("Content-Type")).Equals("application/x-www-form-urlencoded")

	bytes, err := ioutil.ReadAll(client.Requests[0].Body)
	Expect(err).IsNil()
	values, err := url.ParseQuery(string(bytes))
	Expect(err).IsNil()
	Expect(values).HasLen(6)
	Expect(values.Get("to")).Equals(`"Jon Sow" <jon.snow@got.com>`)
	Expect(values.Get("from")).Equals(`"Fider Test" <noreply@random.org>`)
	Expect(values.Get("h:Reply-To")).Equals("noreply@random.org")
	Expect(values.Get("subject")).Equals("Message to: Hello")
	Expect(values["o:tag"][0]).Equals("template:echo_test")
	Expect(values["o:tag"][1]).Equals("tenant:got")
	Expect(values.Get("html")).Equals(`<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">
<html>
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
		<meta name="viewport" content="width=device-width">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
	</head>
	<body bgcolor="#F7F7F7" style="font-size:16px">
		<table width="100%" bgcolor="#F7F7F7" cellpadding="0" cellspacing="0" border="0" style="text-align:center;font-size:14px;">
			<tr>
				<td height="40">&nbsp;</td>
			</tr>
			
			<tr>
				<td align="center">
					<table bgcolor="#FFFFFF" cellpadding="0" cellspacing="0" border="0" style="text-align:left;padding:20px;margin:10px;border-radius:5px;color:#1c262d;border:1px solid #ECECEC;min-width:320px;max-width:660px;">
						Hello World Hello!
					</table>
				</td>
			</tr>
			<tr>
				<td>
					<span style="color:#666;font-size:11px">This email was sent from a notification-only address that cannot accept incoming email. Please do not reply to this message.</span>
				</td>
			</tr>
			<tr>
				<td height="40">&nbsp;</td>
			</tr>
		</table>
	</body>
</html>`)
}

func TestSend_SkipEmptyAddress(t *testing.T) {
	RegisterT(t)
	client.Reset()

	to := email.Recipient{
		Name:    "Jon Sow",
		Address: "",
	}
	sender.Send(ctx, "echo_test", email.Params{
		"name": "Hello",
	}, "Fider Test", to)

	Expect(client.Requests).HasLen(0)
}

func TestSend_SkipUnlistedAddress(t *testing.T) {
	RegisterT(t)
	client.Reset()
	email.SetWhitelist("^.*@gmail.com$")

	to := email.Recipient{
		Name:    "Jon Sow",
		Address: "jon.snow@got.com",
	}
	sender.Send(ctx, "echo_test", email.Params{
		"name": "Hello",
	}, "Fider Test", to)

	Expect(client.Requests).HasLen(0)
}

func TestBatch_Success(t *testing.T) {
	RegisterT(t)
	client.Reset()
	email.SetWhitelist("")

	to := []email.Recipient{
		email.Recipient{
			Name:    "Jon Sow",
			Address: "jon.snow@got.com",
			Params: email.Params{
				"name": "Jon",
			},
		},
		email.Recipient{
			Name:    "Arya Stark",
			Address: "arya.start@got.com",
			Params: email.Params{
				"name": "Arya",
			},
		},
	}
	sender.BatchSend(ctx, "echo_test", email.Params{}, "Fider Test", to)

	Expect(client.Requests).HasLen(1)
	Expect(client.Requests[0].URL.String()).Equals("https://api.mailgun.net/v3/mydomain.com/messages")
	Expect(client.Requests[0].Header.Get("Authorization")).Equals("Basic YXBpOm15czNjcjN0azN5")
	Expect(client.Requests[0].Header.Get("Content-Type")).Equals("application/x-www-form-urlencoded")

	bytes, err := ioutil.ReadAll(client.Requests[0].Body)
	Expect(err).IsNil()
	values, err := url.ParseQuery(string(bytes))
	Expect(err).IsNil()
	Expect(values).HasLen(7)
	Expect(values["to"]).HasLen(2)
	Expect(values["to"][0]).Equals(`"Jon Sow" <jon.snow@got.com>`)
	Expect(values["to"][1]).Equals(`"Arya Stark" <arya.start@got.com>`)
	Expect(values.Get("from")).Equals(`"Fider Test" <noreply@random.org>`)
	Expect(values.Get("h:Reply-To")).Equals("noreply@random.org")
	Expect(values.Get("subject")).Equals("Message to: %recipient.name%")
	Expect(values["o:tag"]).HasLen(2)
	Expect(values["o:tag"][0]).Equals("template:echo_test")
	Expect(values["o:tag"][1]).Equals("tenant:got")
	Expect(values.Get("recipient-variables")).Equals("{\"arya.start@got.com\":{\"name\":\"Arya\"},\"jon.snow@got.com\":{\"name\":\"Jon\"}}")
	Expect(values.Get("html")).Equals(`<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">
<html>
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
		<meta name="viewport" content="width=device-width">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
	</head>
	<body bgcolor="#F7F7F7" style="font-size:16px">
		<table width="100%" bgcolor="#F7F7F7" cellpadding="0" cellspacing="0" border="0" style="text-align:center;font-size:14px;">
			<tr>
				<td height="40">&nbsp;</td>
			</tr>
			
			<tr>
				<td align="center">
					<table bgcolor="#FFFFFF" cellpadding="0" cellspacing="0" border="0" style="text-align:left;padding:20px;margin:10px;border-radius:5px;color:#1c262d;border:1px solid #ECECEC;min-width:320px;max-width:660px;">
						Hello World %recipient.name%!
					</table>
				</td>
			</tr>
			<tr>
				<td>
					<span style="color:#666;font-size:11px">This email was sent from a notification-only address that cannot accept incoming email. Please do not reply to this message.</span>
				</td>
			</tr>
			<tr>
				<td height="40">&nbsp;</td>
			</tr>
		</table>
	</body>
</html>`)
}

func TestGetBaseURL(t *testing.T) {
	RegisterT(t)

	// Fall back to US if there is nothing set
	env.Config.Email.Mailgun.Region = ""
	Expect(sender.GetBaseURL()).Equals("https://api.mailgun.net/v3/mydomain.com/messages")
	
	// Return the EU domain for EU, ignore the case
	env.Config.Email.Mailgun.Region = "EU"
	Expect(sender.GetBaseURL()).Equals("https://api.eu.mailgun.net/v3/mydomain.com/messages")
	env.Config.Email.Mailgun.Region = "eu"
	Expect(sender.GetBaseURL()).Equals("https://api.eu.mailgun.net/v3/mydomain.com/messages")

	// Return the US domain for US, ignore the case
	env.Config.Email.Mailgun.Region = "US"
	Expect(sender.GetBaseURL()).Equals("https://api.mailgun.net/v3/mydomain.com/messages")
	env.Config.Email.Mailgun.Region = "us"
	Expect(sender.GetBaseURL()).Equals("https://api.mailgun.net/v3/mydomain.com/messages")

    // Return the US domain if the region is invalid
	env.Config.Email.Mailgun.Region = "Mars"
	Expect(sender.GetBaseURL()).Equals("https://api.mailgun.net/v3/mydomain.com/messages")

}