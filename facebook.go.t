package main

import "golang.org/x/oauth2"

var FACEBOOK oauth2.Config

type FacebookConfigType struct {
  ClientID string
  ClientSecret string
}

var FacebookConfig *FacebookConfigType

func (c Facebook) AuthoriseFacebook() revel.Result {
    authUrl := FACEBOOK.AuthCodeURL(FACEBOOK.RedirectURL)
    return c.Redirect(authUrl)
}

func (c Facebook) Save(code string) revel.Result {
    var tok *oauth2.Token
    var err error
    var user *models.User
    tok, err = FACEBOOK.Exchange(nil, code)
        facebookAccount = models.FacebookAccount{AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken, UserId:user.Id, Expiry : tok.Expiry.UnixNano()}
    }

    c.Begin()
    defer func() {
        if (c.Txn != nil) {
            c.Rollback()
        }
    }()
    c.Txn.Save(&facebookAccount)
    c.Commit()
}

func (c Facebook) Post() revel.Result {
    httpClient := FACEBOOK.Client(nil, &oauth2.Token{AccessToken:facebookAccount.AccessToken, Expiry: time.Unix(0,facebookAccount.Expiry)})
    httpClient.Post("https://graph.facebook.com/me/feed?message={urlencodedmessage}&access_token="+ facebookAccount.AccessToken, "text/plain", bytes.NewBufferString(""))
    return c.RenderText("Done")
}
