package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/machinebox/graphql"
	"log"
	"os"
)

var APITO_TOKEN = os.Getenv("APITO_TOKEN")

type User struct {
	Id   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

type Users struct {
	Users []*User `json:"users"`
}

type UpdateUser struct {
	UpdateUser *User `json:"updateUser"`
}


// HandleRequest will be called when the lambda function is invoked
// it takes an input and checks if it matches our super secret value
func HandleRequest(ctx context.Context, input json.RawMessage) (interface{}, error) {

	var inputMessage map[string]interface{}
	err := json.Unmarshal(input, &inputMessage)
	if err != nil {
		return nil, errors.New("Input Json Unmarshal Error :" + err.Error())
	}

	log.Print("Input Message : ")
	log.Println(inputMessage) // Store the Input Message as Lambda Logs

	if phone, ok := inputMessage["phone"].(string); ok {
		// create a GraphQL client (safe to share across requests)
		client := graphql.NewClient("https://api.apito.io/graphql")

		// make a request
		req := graphql.NewRequest(`
		   query FindUser($phone: String) {
			  users(where: {phone: {eq: $phone}}) {
				id
				data {
				  phone
				}
			  }
			}`)

		// set any variables
		req.Var("phone", phone)

		// set header fields
		req.Header.Set("Authorization", "Bearer "+APITO_TOKEN)

		// run it and capture the response
		var respData Users
		if err := client.Run(ctx, req, &respData); err != nil {
			return nil, err
		}

		if respData.Users != nil {
			// unwrap the data
			user := respData.Users[0]
			if user != nil {
				// user found, Send a OTP Request via your SMS provider
				otp := "1234" // generate it randomly

				// ---- YOUR SMS PROVIDER CODE GOES HERE -----

				// Save the User OTP as User Secret
				req := graphql.NewRequest(`
					   mutation UseOTPAsUserSecret($id: String!, $otp: String) {
						  updateUser(_id: $id, payload: {secret: $otp}) {
							id
							data {
							  phone
							}
						  }
						}`)
				// set any variables
				req.Var("id", user.Id)
				req.Var("otp", otp)

				// set header fields
				req.Header.Set("Authorization", "Bearer "+APITO_TOKEN)

				// run it and capture the response
				var updateUserResponse UpdateUser
				if err := client.Run(ctx, req, &updateUserResponse); err != nil {
					return nil, err
				}

				log.Print("Update User : ")
				fmt.Println(updateUserResponse)
				return map[string]interface{}{
					"send_otp" : map[string]interface{} { // apito function name
						"user": updateUserResponse.UpdateUser, // because response type is user
					},
				}, nil
			}
		} else {
			return nil, errors.New("Could Not Find any User Using :" + inputMessage["phone"].(string))
		}
	}

	return nil, errors.New("Phone number not found invalid request")
}

func main() {
	lambda.Start(HandleRequest)
	// For Local testing,
	/*resp, err := HandleRequest(context.Background(), []byte(`{ "phone" : "01760000000"}`))
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(resp)*/
}
