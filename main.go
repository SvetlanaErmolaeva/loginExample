package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

var (
	tmpl          = template.Must(template.ParseGlob("templates/*.html"))
	clientID      = "320"
	clientSecret  = "JzJkTOYS9WVEM6zBkTRCNC4cGmbBXOM8"
	redirectURI   = "http://localhost:8080/token"
	state         = "CUSTOM_STATE"
	response_type = "code"
	claims        map[string]interface{}
)

type page struct {
	Login      string
	Registered string
	User       string
	Tag        string
}

func main() {
	http.HandleFunc("/", index)
	http.HandleFunc("/token", getToken)
	http.HandleFunc("/auth", auth)
	log.Println("-> Server has started")
	log.Print(http.ListenAndServe(":8080", nil))
	log.Println("-> Server has stopped")
}

func auth(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("https://login.xsolla.com/api/oauth2/login?response_type=%s&client_id=%s&state=%s&redirect_uri=%s", response_type, clientID, state, redirectURI)

	err := tmpl.ExecuteTemplate(w, "auth.html", url)
	if err != nil {
		log.Printf("error = %s", err)
	}
}
func getToken(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("token")

	// decode JWT token and verify signature using JSON Web Keyset
	token, _ := jwt.Parse(t, nil)
	if token == nil {
		fmt.Printf("error")
	}
	claims, _ := token.Claims.(jwt.MapClaims)
	for key, value := range claims {
		fmt.Printf("%s=%s\n", key, value)
	}
	sub := claims["sub"]
	userUrl := fmt.Sprintf("https://login.xsolla.com/api/users/%s/public", sub)
	res, _ := http.NewRequest("GET", userUrl, nil)
	res.Header.Add("Authorization", t)
	resp, _ := http.DefaultClient.Do(res)
	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	var dat map[string]string
	if err := json.Unmarshal(bodyBytes, &dat); err != nil {
		panic(err)
	}

	page := page{Login: dat["last_login"], Registered: dat["registered"], Tag: dat["tag"], User: dat["user_id"]}
	tmpl.ExecuteTemplate(w, "me.html", page)
}

func refresh(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("https://login.xsolla.com/api/oauth2/token")
	values := map[string]string{"grant_type": "refresh_token",
		"client_id": clientID, "redirect_uri": redirectURI,
		"client_secret": clientSecret}
	json_data, err := json.Marshal(values)
	if err != nil {
		log.Fatal("Error in requst")
		log.Fatal(err)
	}

	request, err := http.NewRequest("http.MethodPost", url, bytes.NewBuffer(json_data))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	respo, _ := http.DefaultClient.Do(request)
	fmt.Printf(respo.Status)
}

func index(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("https://login.xsolla.com/api/oauth2/login?response_type=%s&client_id=%s&state=%s", response_type, clientID, state)
	err := tmpl.ExecuteTemplate(w, "index.html", url)
	if err != nil {
		log.Printf("error = %s", err)
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	for key, value := range r.Form {
		fmt.Printf("%s=%s\n", key, value)
	}
	url := fmt.Sprintf("https://login.xsolla.com/api/oauth2/login?response_type=code&client_id=%s&state=%s", clientID, state)
	values := map[string]string{"password": "12345678", "username": "ermolaevasn13@gmail.com"}
	json_data, err := json.Marshal(values)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(json_data))

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	fmt.Println(res["login_url"])
}

func me(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("token")
	fmt.Printf("TOKEN:")
	fmt.Printf(t)
	stateTemp := r.URL.Query().Get("state")
	if stateTemp[len(stateTemp)-1] == '}' {
		stateTemp = stateTemp[:len(stateTemp)-1]
	}
	if stateTemp == "" {
		respErr(w, fmt.Errorf("state query param is not provided"))
		return
	} else if stateTemp != state {
		respErr(w, fmt.Errorf("state query param do not match original one, got=%s", stateTemp))
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		respErr(w, fmt.Errorf("code query param is not provided"))
		return
	}
	fmt.Printf("CODE:")
	fmt.Printf(code)

	url := fmt.Sprintf("https://login.xsolla.com/api/oauth2/token")
	values := map[string]string{"grant_type": "authorization_code",
		"code": code, "client_id": clientID, "redirect_uri": redirectURI,
		"client_secret": clientSecret}
	json_data, err := json.Marshal(values)
	if err != nil {
		log.Fatal("Error in requst")
		log.Fatal(err)
	}

	res, err := http.NewRequest("http.MethodPost", url, bytes.NewBuffer(json_data))
	res.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&res)
	fmt.Println(result["access_token"])

	resp, err := http.DefaultClient.Do(res)
	if err != nil {
		respErr(w, err)
		fmt.Printf("error")
		return
	}
	fmt.Println(resp.Status)
	defer resp.Body.Close()
	token := struct {
		AccessToken string `json:"access_token"`
	}{}
	bytes, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(bytes, &token)

}

func respErr(w http.ResponseWriter, err error) {
	_, er := io.WriteString(w, err.Error())
	if er != nil {
		log.Println(err)
	}
}
