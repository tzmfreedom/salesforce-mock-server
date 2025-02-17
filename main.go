package main

import (
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"os"
)

type LoginEnvelope struct {
	Body struct {
		Login struct {
			Username string `xml:"username"`
			Password string `xml:"password"`
		} `xml:"login"`
	} `xml:"Body"`
}

type LoginResponse struct {
	XMLName     xml.Name    `xml:"urn:partner.soap.sforce.com loginResponse"`
	LoginResult LoginResult `xml:"result"`
}

type LoginResult struct {
	MetadataServerUrl string `xml:"metadataServerUrl"`
	PasswordExpired   bool   `xml:"passwordExpired"`
	Sandbox           bool   `xml:"sandbox""`
	ServerUrl         string `xml:"serverUrl"`
	SessionId         string `xml:"sessionId"`
	//UserId            *ID    `xml:"userId"`
}

var data = map[string][]Record{}

type Record struct {
	Fields map[string]interface{}
}

var sessionIds = map[string]struct{}{}

func main() {
	server := http.Server{
		Addr:    ":8080",
		Handler: nil,
	}
	username := os.Getenv("SF_USERNAME")
	password := os.Getenv("SF_PASSWORD")

	http.HandleFunc("/services/Soap/u/{version}/", func(w http.ResponseWriter, r *http.Request) {
		var result *LoginEnvelope
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		err = xml.Unmarshal(buf, &result)
		if err != nil {
			panic(err)
		}
		if result.Body.Login.Username != username || result.Body.Login.Password != password {
			w.Write([]byte("ng"))
		} else {
			sessionId := rand.Text()
			sessionIds[sessionId] = struct{}{}
			res := LoginResponse{
				XMLName: xml.Name{},
				LoginResult: LoginResult{
					MetadataServerUrl: "",
					PasswordExpired:   false,
					Sandbox:           false,
					ServerUrl:         "",
					SessionId:         sessionId,
				},
			}
			buf, err := xml.Marshal(res)
			if err != nil {
				panic(err)
			}
			w.Write(buf)
		}
	})
	http.HandleFunc("/services/data/{version}/query", func(w http.ResponseWriter, r *http.Request) {
		//q := r.URL.Query().Get("q")
		//_, _ = w.Write([]byte(r.PathValue("version")))
		buf, err := json.Marshal(data)
		if err != nil {
			panic(err)
		}
		w.Write(buf)
	})
	http.HandleFunc("/services/data/{version}/sobjects/{sobject}/", func(w http.ResponseWriter, r *http.Request) {
		var fields map[string]interface{}
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(buf, &fields)
		if err != nil {
			panic(err)
		}
		sobjectKey := r.PathValue("sobject")
		records := data[sobjectKey]
		data[sobjectKey] = append(records, Record{Fields: fields})
	})
	http.HandleFunc("/_reset", func(w http.ResponseWriter, r *http.Request) {
		data = map[string][]Record{}
		sessionIds = map[string]struct{}{}
	})

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
