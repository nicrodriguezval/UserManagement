package main

import (
	"UserManagementMS/DBConnection"
	"UserManagementMS/Encryption"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type User struct {
	ID             primitive.ObjectID  `json:"_id,omitempty" bson:"_id,omitempty"`
	Firstname      string              `json:"firstname,omitempty" bson:"firstname,omitempty"`
	Lastname       string              `json:"lastname,omitempty" bson:"lastname,omitempty"`
	Username       string              `json:"username,omitempty" bson:"username,omitempty"`
	Password       string              `json:"password,omitempty" bson:"password,omitempty"`
	Country        string              `json:"country,omitempty" bson:"country,omitempty"`
	ProfilePicture string              `json:"profile_picture,omitempty" bson:"profile_picture,omitempty"`
	CreatedAt      primitive.Timestamp `json:"created_at,omitempty" bson:"created_at,omitempty"`
}

var client *mongo.Client

func CreateUserEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "application/json")
	var user User
	_ = json.NewDecoder(req.Body).Decode(&user)
	user.Password = string(Encryption.Encrypt([]byte(user.Password), "password"))
	collection := client.Database("geosmart_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	result, err := collection.InsertOne(ctx, user)
	if err != nil {
		log.Fatal(err)
	}
	res.WriteHeader(http.StatusCreated)
	json.NewEncoder(res).Encode(result)
}

func GetUsersEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "application/json")
	var users []User
	collection := client.Database("geosmart_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "` + err.Error() + `"}`))
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var user User
		cursor.Decode(&user)
		users = append(users, user)
	}
	if err := cursor.Err(); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "` + err.Error() + `"}`))
		return
	}
	json.NewEncoder(res).Encode(users)
}

func GetUserEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "application/json")
	params := mux.Vars(req)
	id, _ := primitive.ObjectIDFromHex(params["id"])
	var user User
	collection := client.Database("geosmart_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := collection.FindOne(ctx, User{ID: id}).Decode(&user)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "` + err.Error() + `"}`))
		return
	}
	json.NewEncoder(res).Encode(user)
}

func DeleteUserEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "application/json")
	params := mux.Vars(req)
	id, _ := primitive.ObjectIDFromHex(params["id"])
	var user User
	collection := client.Database("geosmart_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := collection.FindOneAndDelete(ctx, User{ID: id}).Decode(&user)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "` + err.Error() + `"}`))
		return
	}
	json.NewEncoder(res).Encode(user)
}

func LoginUserEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "appication/json")
	var user User
	var result User
	_ = json.NewDecoder(req.Body).Decode(&user)
	collection := client.Database("geosmart_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := collection.FindOne(ctx, User{Username: user.Username}).Decode(&result)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte(`{ "message": "User with username ` + user.Username + ` doesn't exist" }`))
		return
	}
	if user.Password != string(Encryption.Decrypt([]byte(result.Password), "password")) {
		res.WriteHeader(http.StatusUnauthorized)
		res.Write([]byte(`{ "message": "Wrong password" }`))
		return
	}
	json.NewEncoder(res).Encode(result)
}

func UpdateuserEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "appication/json")
	params := mux.Vars(req)
	id, _ := primitive.ObjectIDFromHex(params["id"])
	var user User
	_ = json.NewDecoder(req.Body).Decode(&user)
	collection := client.Database("geosmart_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	result, err := collection.UpdateOne(ctx, bson.M{"_id": id}, bson.D{{"$set", user}})
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte(`{ "message": "User doesn't exist" }`))
		return
	}
	json.NewEncoder(res).Encode(result)
}

func main() {
	// database connection
	var ctx context.Context
	client, ctx = DBConnection.Connection()
	defer client.Disconnect(ctx)

	// all routes for API REST
	router := mux.NewRouter()
	router.HandleFunc("/user", CreateUserEndpoint).Methods("POST")
	router.HandleFunc("/user/login", LoginUserEndpoint).Methods("POST")
	router.HandleFunc("/user", GetUsersEndpoint).Methods("GET")
	router.HandleFunc("/user/{id}", GetUserEndpoint).Methods("GET")
	router.HandleFunc("/user/{id}", DeleteUserEndpoint).Methods("DELETE")
	router.HandleFunc("/user/{id}", UpdateuserEndpoint).Methods("PUT")

	// port listening
	log.Fatal(http.ListenAndServe(":3000", router))
}
