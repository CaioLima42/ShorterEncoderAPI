package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/mux"
)

type HandleViaStruct struct{}

type Store interface{
	Add(shortenedUrl, longUrl string) error
	Remove(shortenedUrl string) error
	Get(shortenedUrl string) (string, error)
}

type MemoryStore struct{
	items map[string]string
}
func NewMemoryStore()*MemoryStore{
	mem := new(MemoryStore)
	mem.items = make(map[string]string)
	return mem
}

func (m *MemoryStore) Add(shortenedUrl, longUrl string) error{
	if _, ok := m.items[shortenedUrl] ; ok{
		return fmt.Errorf("esta Url já esta sendo usada")
	} 
	m.items[shortenedUrl] = longUrl
	log.Println(m.items)
	return nil
}

func (m * MemoryStore) Remove(shortenedUrl string) error{
	if _,ok := m.items[shortenedUrl]; !ok{
		return fmt.Errorf("esta url não pode ser revolvida pois ela não existe")
	}
	delete(m.items, shortenedUrl)
	log.Printf("Url %s remolvida com sucesso", shortenedUrl)
	return nil
}

func (m * MemoryStore) Get(shortenedUrl string) (string, error){
	v, ok := m.items[shortenedUrl]; 
	if !ok{
		return "", fmt.Errorf("esta url não foi encontrada")
	}
	return v , nil
}

func (*HandleViaStruct) ServeHTTP(w http.ResponseWriter,r *http.Request){
	log.Print("Start Hello world")
	defer log.Print("End Hello world")
	fmt.Fprintf(w, "Hello world")
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
	r := mux.NewRouter()
	redirectPath := "localhost:8080/r"
	mem := NewMemoryStore()
	r.Handle("/", &HandleViaStruct{}).Methods("GET")
	r.Handle("/add", &AddPath{domain: redirectPath, store: mem}).Methods("POST")
	r.Handle("/r/{hash}", &DeletePath{mem}).Methods("DELETE")
	r.Handle("/r/{hash}", &RedirectPath{mem}).Methods("GET") 
	log.Fatal(http.ListenAndServe(":8080", r))
	
}


