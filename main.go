package main

import (
  "encoding/json"
  "fmt"
  "log"
  "net/http"
  "os"
  "strconv"
  "strings"
  "time"

  "github.com/gorilla/handlers"
  "github.com/gorilla/mux"
  "github.com/jsgoecke/tesla"
)

// Secret to prove you are worthy.
var Secret = os.Getenv("SECRET")

type request struct {
  Secret string `json:"secret"`
  State  string `json:"state"`
}

func secretOk(request request) bool {
  if request.Secret == Secret {
    return true
  }
  return false
}

func root(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("Hello World!\n"))
}

func party(w http.ResponseWriter, r *http.Request) {
  var request request
  _ = json.NewDecoder(r.Body).Decode(&request)
  if !secretOk(request) {
    w.WriteHeader(http.StatusForbidden)
    w.Write([]byte("nope\n"))
    return
  }
  if strings.ToLower(request.State) == "on" {
    go func() {
      _, _ = http.Get("http://192.168.1.247/rotate")
    }()
  }
  if strings.ToLower(request.State) == "off" {
    go func() {
      _, _ = http.Get("http://192.168.1.247/setcolor?red=255&green=255&blue=255")
    }()
  }
  w.Write([]byte("ok\n"))
}

func hotwater(w http.ResponseWriter, r *http.Request) {
  var request request
  _ = json.NewDecoder(r.Body).Decode(&request)
  if !secretOk(request) {
    w.WriteHeader(http.StatusForbidden)
    w.Write([]byte("nope\n"))
    return
  }
  if strings.ToLower(request.State) == "on" {
    go func() {
      _, _ = http.Get("http://192.168.1.247/setcolor?red=255&green=0&blue=0")
      _, _ = http.Get("http://192.168.1.150:8080/HW/on?for=900")
      time.Sleep(1 * time.Second)
      _, _ = http.Get("http://192.168.1.247/setcolor?red=255&green=255&blue=255")
    }()
  }
  if strings.ToLower(request.State) == "off" {
    go func() {
      _, _ = http.Get("http://192.168.1.247/setcolor?red=0&green=0&blue=255")
      _, _ = http.Get("http://192.168.1.150:8080/HW/off")
      time.Sleep(1 * time.Second)
      _, _ = http.Get("http://192.168.1.247/setcolor?red=255&green=255&blue=255")
    }()
  }
  w.Write([]byte("ok\n"))
}

func conditionTesla(w http.ResponseWriter, r *http.Request) {
  var request request
  _ = json.NewDecoder(r.Body).Decode(&request)
  if !secretOk(request) {
    w.WriteHeader(http.StatusForbidden)
    w.Write([]byte("nope\n"))
    return
  }
  go func() {
    body := "{\"animation\": \"cylon\", \"rgbw\": \"0,255,0,0\", \"percent\": 10.0, \"velocity\" : 30}"
    httpClient := &http.Client{}
    req, _ := http.NewRequest(http.MethodPut, "http://192.168.1.127:9000/lumen", strings.NewReader(body))
    _, _ = httpClient.Do(req)
    client, err := tesla.NewClient(
      &tesla.Auth{
        ClientID:     os.Getenv("TESLA_CLIENT_ID"),
        ClientSecret: os.Getenv("TESLA_CLIENT_SECRET"),
        Email:        os.Getenv("TESLA_USERNAME"),
        Password:     os.Getenv("TESLA_PASSWORD"),
      })
    if err != nil {
      panic(err)
    }
    vehicles, err := client.Vehicles()
    if err != nil {
      panic(err)
    }

    vehicle := vehicles[0]
    fmt.Println(vehicle.State)
    var count int = 0
    state, _ := vehicle.Wakeup()
    for state.State != "online" {
      count++
      fmt.Println(state.State + " " + strconv.Itoa(count))
      time.Sleep(3 * time.Second)
      state, _ = vehicle.Wakeup()
      if count >= 30 {
        body := "{\"animation\": \"cylon\", \"rgbw\": \"0,0,255,0\", \"percent\": 10.0, \"velocity\" : 30}"
        httpClient := &http.Client{}
        req, _ := http.NewRequest(http.MethodPut, "http://192.168.1.127:9000/lumen", strings.NewReader(body))
        _, _ = httpClient.Do(req)
        return
      }
    }
    fmt.Println(state.State)

    if strings.ToLower(request.State) == "on" {
      _ = vehicle.StartAirConditioning()
    }
    if strings.ToLower(request.State) == "off" {
      _ = vehicle.StopAirConditioning()
    }
    _ = vehicle.FlashLights()
  }()

}

func main() {
  r := mux.NewRouter()
  r.HandleFunc("/", root)
  r.HandleFunc("/kitchen/party", party)
  r.HandleFunc("/hotwater", hotwater)
  r.HandleFunc("/tesla/condition", conditionTesla)

  loggedRouter := handlers.LoggingHandler(os.Stdout, r)
  log.Fatal(http.ListenAndServe(":8000", loggedRouter))
}
