package main

import (
	"UserManagementMS/Auth"
	"UserManagementMS/DBConnection"
	"UserManagementMS/Encryption"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/gorilla/mux"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type User struct {
	ID             primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Firstname      string             `json:"firstname,omitempty" bson:"firstname,omitempty"`
	Lastname       string             `json:"lastname,omitempty" bson:"lastname,omitempty"`
	Username       string             `json:"username,omitempty" bson:"username,omitempty"`
	Password       string             `json:"password,omitempty" bson:"password,omitempty"`
	Country        string             `json:"country,omitempty" bson:"country,omitempty"`
	ProfilePicture string             `json:"profile_picture,omitempty" bson:"profile_picture,omitempty"`
	CreatedAt      time.Time          `json:"created_at,omitempty" bson:"created_at,omitempty"`
	Flag           string             `json:"flag,omitempty" bson:"flag,omitempty"`
}

type NewUser struct {
	ID             primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Firstname      string             `json:"firstname,omitempty" bson:"firstname,omitempty"`
	Lastname       string             `json:"lastname,omitempty" bson:"lastname,omitempty"`
	Username       string             `json:"username,omitempty" bson:"username,omitempty"`
	Password       string             `json:"password,omitempty" bson:"password,omitempty"`
	NewPassword    string             `json:"new_password,omitempty" bson:"new_password,omitempty"`
	Country        string             `json:"country,omitempty" bson:"country,omitempty"`
	ProfilePicture string             `json:"profile_picture,omitempty" bson:"profile_picture,omitempty"`
	CreatedAt      time.Time          `json:"created_at,omitempty" bson:"created_at,omitempty"`
	Flag           string             `json:"flag,omitempty" bson:"flag,omitempty"`
}

type Guest struct {
	Username string `json:"username,omitempty" bson:"username,omitempty"`
}

var client *mongo.Client

const ldapserver = "ldap://18.210.193.21"

func CreateUserEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "application/json")
	var user User
	var dbuser User
	_ = json.NewDecoder(req.Body).Decode(&user)
	userpassword := user.Password
	user.Password = string(Encryption.Encrypt([]byte(userpassword), "password"))
	collection := client.Database("UserManagement_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_ = collection.FindOne(ctx, bson.M{"username": user.Username}).Decode(&dbuser)

	if user.Username == dbuser.Username {
		res.WriteHeader(http.StatusConflict)
		res.Write([]byte(`{ "message": "username ` + dbuser.Username + ` already exist"}`))
		return
	}

	user.CreatedAt = time.Now()

	_, err := collection.InsertOne(ctx, user)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "` + err.Error() + `"}`))
		return
	}

	// executing create user command for ldpa
	cmd := exec.Command(`ldapadd -H ` + ldapserver + ` -D "cn=admin,dc=swarch,dc=geosmart,dc=com" -w "admin"\n` +
		`dn: uid=` + user.Username + `,ou=development,dc=swarch,dc=geosmart,dc=com\n` +
		`objectClass: top\n` +
		`objectclass: inetOrgPerson\n` +
		`objectClass: posixAccount\n` +
		`gn:` + user.Firstname + `\n` +
		`sn:` + user.Lastname + `\n` +
		`cn:` + user.Username + `@unal.edu.co\n` +
		`uid:` + user.Username + `\n` +
		`uidNumber: 1000\n` +
		`gidNumber: 500\n` +
		`homeDirectory: /home/` + user.Username + `\n` +
		`loginShell: /bin/bash\n` +
		`userPassword: {crypt}x`)

	if err := cmd.Run(); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "` + err.Error() + `"}`))
		return
	}

	// update password for encryption
	cmd = exec.Command(`ldappasswd -H ` + ldapserver + ` -D "cn=admin,dc=swarch,dc=geosmart,dc=com" -w "admin" "uid=` + user.Username + `,ou=development,dc=swarch,dc=geosmart,dc=com" -s ` + userpassword)
	if err := cmd.Run(); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "` + err.Error() + `"}`))
		return
	}

	res.WriteHeader(http.StatusCreated)
	json.NewEncoder(res).Encode(bson.M{"message": "Successfully created user"})
}

func GetUsersEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "application/json")
	var users []User
	collection := client.Database("UserManagement_db").Collection("User")
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
	collection := client.Database("UserManagement_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)

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
	var user User
	id, _ := primitive.ObjectIDFromHex(params["id"])
	collection := client.Database("UserManagement_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := collection.FindOneAndDelete(ctx, bson.M{"_id": id}).Decode(&user)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "` + err.Error() + `"}`))
		return
	}

	json.NewEncoder(res).Encode(bson.M{"message": "user with username " + user.Username + " successfully deleted"})
}

func LoginUserEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "appication/json")
	var user User
	var result User
	_ = json.NewDecoder(req.Body).Decode(&user)
	collection := client.Database("UserManagement_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := collection.FindOne(ctx, bson.M{"username": user.Username}).Decode(&result)

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

	var tokenString string
	tokenString, err = Auth.GenerateJWT(true, result.ID.Hex())

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "something went wrong: ` + err.Error() + `" }`))
		return
	}

	json.NewEncoder(res).Encode(bson.M{
		"_id":             result.ID,
		"firstname":       result.Firstname,
		"lastname":        result.Lastname,
		"username":        result.Username,
		"country":         result.Country,
		"profile_picture": result.ProfilePicture,
		"created_at":      result.CreatedAt,
		"flag":            result.Flag,
		"token":           tokenString,
	})
}

func UpdateuserEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "appication/json")
	params := mux.Vars(req)
	id, _ := primitive.ObjectIDFromHex(params["id"])
	var newUser NewUser
	var result User
	_ = json.NewDecoder(req.Body).Decode(&newUser)
	collection := client.Database("UserManagement_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 15*time.Second)
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&result)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "` + err.Error() + `"}`))
		return
	}

	if newUser.Password != string(Encryption.Decrypt([]byte(result.Password), "password")) {
		res.WriteHeader(http.StatusUnauthorized)
		res.Write([]byte(`{ "message": "Wrong password" }`))
		return
	}

	var user User

	user.Firstname = newUser.Firstname
	user.Lastname = newUser.Lastname
	user.Username = newUser.Username
	if len(newUser.NewPassword) > 0 {
		user.Password = string(Encryption.Encrypt([]byte(newUser.NewPassword), "password"))
	}
	user.Country = newUser.Country
	user.ProfilePicture = newUser.ProfilePicture
	user.Flag = newUser.Flag

	_, err = collection.UpdateOne(ctx, bson.M{"_id": id}, bson.D{{"$set", user}})

	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte(`{ "message": "User doesn't exist" }`))
		return
	}

	var resultUser User
	err = collection.FindOne(ctx, bson.M{"_id": id}).Decode(&resultUser)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "` + err.Error() + `"}`))
	}

	json.NewEncoder(res).Encode(bson.M{
		"_id":             resultUser.ID,
		"firstname":       resultUser.Firstname,
		"lastname":        resultUser.Lastname,
		"username":        resultUser.Username,
		"country":         resultUser.Country,
		"profile_picture": resultUser.ProfilePicture,
		"created_at":      resultUser.CreatedAt,
		"flag":            resultUser.Flag,
	})
}

func LoginGuestEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "application/json")
	var guest Guest
	var dbuser User
	_ = json.NewDecoder(req.Body).Decode(&guest)
	collection := client.Database("UserManagement_db").Collection("User")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_ = collection.FindOne(ctx, bson.M{"username": guest.Username}).Decode(&dbuser)

	if guest.Username == dbuser.Username {
		res.WriteHeader(http.StatusConflict)
		res.Write([]byte(`{ "message": "username ` + dbuser.Username + ` already exist"}`))
		return
	}

	tokenString, err := Auth.GenerateJWT(false, guest.Username)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "something went wrong: ` + err.Error() + `" }`))
		return
	}

	json.NewEncoder(res).Encode(bson.M{"token": tokenString})
}

func ValidateTokenEndpoint(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", "application/json")
	params := mux.Vars(req)
	tokenString, _ := params["token"]
	tkn, err := Auth.VerifyToken(tokenString)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{ "message": "` + err.Error() + `"}`))
		return
	}

	if tkn != nil {
		json.NewEncoder(res).Encode(bson.M{"valid": true})
	} else {
		json.NewEncoder(res).Encode(bson.M{"valid": false})
	}
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
	router.HandleFunc("/guest/login", LoginGuestEndpoint).Methods("POST")
	router.HandleFunc("/token/validate-token/{token}", ValidateTokenEndpoint).Methods("GET")

	// port listening
	log.Fatal(http.ListenAndServe(":3000", router))
}
