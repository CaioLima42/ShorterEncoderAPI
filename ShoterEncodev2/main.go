package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"os"
)

type HandleViaStruct struct{}

type Store interface{
	Add(shortenedUrl, longUrl string) error
	Remove(shortenedUrl string) error
	Get(shortenedUrl string) (string, error)
}

type internalStoreStruct struct{
	Version 	string		   `json:"version"`
	Item 		map[string]string `json:"item"`
}


type FileStore struct{
	fileName string
}

func (f *FileStore) Add(shortenedUrl, longUrl string) error{
	data, err := os.ReadFile(f.fileName)
	if err != nil{
		return fmt.Errorf("error so read the file")
	}
	var is internalStoreStruct
	err = json.Unmarshal(data, &is)
	if err != nil{
		return fmt.Errorf("error in json format")
	}
	if _, ok := is.Item[shortenedUrl]; ok{
		return fmt.Errorf("the url arealy exist")
	}
	log.Print(is.Version)
	is.Item[shortenedUrl] = longUrl
	modData, err := json.Marshal(is)
	if err != nil{
		return fmt.Errorf("error on marshal the json")
	}
	err = os.WriteFile(f.fileName, modData, 0644)
	if err != nil{
		return fmt.Errorf("error on write the json file")
	}
	return nil
}

func (f* FileStore)	Remove(shortenedUrl string) error{
	data, err := os.ReadFile(f.fileName)
	if err != nil{
		return fmt.Errorf("error on read json file")
	}

	var is internalStoreStruct
	err = json.Unmarshal(data, &is)
	if err != nil{
		return fmt.Errorf("error on json format sended")
	}
	if _, ok := is.Item[shortenedUrl]; !ok{
		return fmt.Errorf("the url %s not exist", shortenedUrl) 
	}
	delete(is.Item, shortenedUrl)

	modData, err := json.Marshal(is)
	if err != nil{
		return fmt.Errorf("error on marshal the json")
	}
	err = os.WriteFile(f.fileName, modData, 0644)
	if err != nil{
		return fmt.Errorf("error on save the json")
	}
	return nil
}

func (f FileStore) Get(shortenedUrl string) (string, error){
	data, err := os.ReadFile(f.fileName)
	if err != nil{
		return "", fmt.Errorf("error on read the file")
	}
	var is internalStoreStruct
	err = json.Unmarshal(data, &is)
	if err != nil{
		return "", fmt.Errorf("error on json format")
	}
	v, ok := is.Item[shortenedUrl]
	if !ok{
		return "", fmt.Errorf("the url not exist")
	}
	return v, nil
	
}
func NewFileStore(fileName string) (*FileStore, error){
	store := new(FileStore)
	_, err := os.Stat(fileName)
	if os.IsNotExist(err){
		is := internalStoreStruct{"v1", make(map[string]string)}
		raw,err := json.Marshal(is)
		if err != nil{
			return nil, fmt.Errorf("incorect json format")
		}
		err = os.WriteFile(fileName, raw, 0644)
		if err != nil{
			return nil, err
		}
	}
	store.fileName = fileName
	return	store, nil
}


func (*HandleViaStruct) ServeHTTP(w http.ResponseWriter,r *http.Request){
	log.Print("Start Hello world")
	defer log.Print("End Hello world")
	fmt.Fprintf(w, "Hello world")
	log.Print(r.Context().Value(0))
}

type AddPath struct{
	domain string
	store Store
}

func (a *AddPath) ServeHTTP(w http.ResponseWriter, r *http.Request){
	type addPathRequest struct{
		URL string `json:"url"`
	}
	var parsed addPathRequest
	err := json.NewDecoder(r.Body).Decode(&parsed) // decodifica r.Body em parsed
	log.Print(parsed.URL)
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("unexpected error :: %v", err)))
		return
	}
	if resp,err := http.Get(parsed.URL); resp.StatusCode != 200{
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("unexpected error :: %v", err)))
	}

	h := sha1.New() // criando um novo sha1
	h.Write([]byte(parsed.URL)) // inserindo o valor que queremos passar
	sum := h.Sum(nil) // necessario para criar o nosso sha1
	hash := hex.EncodeToString(sum)[:10] // codificando de hexa para string
	err = a.store.Add(hash, parsed.URL)
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("unexpected error :: %v", err)))
		return
	}

	type addPathResponse struct{
		ShortedURL string `json:"shorted_url"`
		LongURL    string `json:"long_url"`
	}

	pathResp := addPathResponse{ShortedURL: fmt.Sprintf("%v/%v", a.domain, hash), LongURL: parsed.URL}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pathResp)
	
}

type DeletePath struct{
	store Store
}

func (d *DeletePath) ServeHTTP(w http.ResponseWriter, r *http.Request){
	hash := mux.Vars(r)["hash"]

	if hash == ""{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Você precisa colocar a chave hash a ser deletada"))
		return
	}
	err := d.store.Remove(hash)
	
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("unexpected error :: %v", err)))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Url deletada com sucesso"))
}

type RedirectPath struct{
	store Store
}

func (rd *RedirectPath) ServeHTTP(w http.ResponseWriter, r *http.Request){
	hash := mux.Vars(r)["hash"]
	if hash == ""{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Voce precisa colocar a chave hash a ser repassada"))
		return
	}
	url, err := rd.store.Get(hash)

	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("unexpected error :: %v", err) ))
		return
	}
	log.Print(url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func main(){
	log.Print("Hello world sample started.")
	fileName := "Data.json"
	r := mux.NewRouter()
	redirectPath := "localhost:8080/r"
	mem, err := NewFileStore(fileName)
	if err != nil{
		log.Print(err.Error())
		panic("unable to create filestore appropriately")
	}

	r.Handle("/", &HandleViaStruct{}).Methods("GET")
	r.Handle("/add", &AddPath{domain: redirectPath, store: mem}).Methods("POST")
	r.Handle("/r/{hash}", &DeletePath{mem}).Methods("DELETE")
	r.Handle("/r/{hash}", &RedirectPath{mem}).Methods("GET") 
	http.ListenAndServe(":8080", r)
	
}

