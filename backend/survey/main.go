package main

import (
	"context"
	"encoding/json"
	"github.com/aws-samples/artillery-survey-poller/db"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Default headers
var CORSHeaders = map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Credentials": "true"}

/*
Handler method
*/
func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	/*
		Based on HTTP method perform corresponding actions, like:
		POST --> Create new survey
		GET --> Get survey details
		PUT --> Vote on an answer
	*/
	switch request.HTTPMethod {
	case "POST":
		survey := db.Survey{}
		err := json.Unmarshal([]byte(request.Body), &survey)
		if err != nil {
			return events.APIGatewayProxyResponse{Headers: CORSHeaders, StatusCode: 400}, err
		}
		uuid, cErr := db.CreateSurvey(survey)
		if cErr != nil {
			return handleError(cErr)
		} else {
			return events.APIGatewayProxyResponse{Headers: CORSHeaders, Body: *uuid, StatusCode: 200}, nil
		}
	case "GET":
		surveyId := request.QueryStringParameters["surveyId"]
		res, err := db.GetSurvey(surveyId)
		if err != nil {
			return handleError(err)
		}

		data, mErr := json.Marshal(res)
		if mErr != nil {
			return handleError(mErr)
		}

		return events.APIGatewayProxyResponse{Headers: CORSHeaders, Body: string(data), StatusCode: 200}, nil
	case "PUT":
		recreate := request.QueryStringParameters["recreate"]
		// Recreate Request?
		if len(recreate) != 0 {
			// Create table if not exists
			_, cErr := db.CreateTableIfNotExists()
			if cErr != nil {
				return handleError(cErr)
			}

			// Reinsert data
			uuid, err := db.RebuildData()
			if err != nil {
				return handleError(err)
			}

			return events.APIGatewayProxyResponse{Headers: CORSHeaders, Body: *uuid, StatusCode: 200}, nil
		} else {
			surveyId := request.QueryStringParameters["surveyId"]
			questionId := request.QueryStringParameters["questionId"]
			answerId := request.QueryStringParameters["answerId"]

			err := db.Vote(surveyId, questionId, answerId)
			if err != nil {
				return handleError(err)
			} else {
				return events.APIGatewayProxyResponse{Headers: CORSHeaders, StatusCode: 200}, nil
			}
		}
	case "DELETE":
		err := db.DeleteTable()
		if err != nil {
			return handleError(err)
		}

		return events.APIGatewayProxyResponse{Headers: CORSHeaders, StatusCode: 204}, nil
	default:
		return events.APIGatewayProxyResponse{Headers: CORSHeaders, StatusCode: 400}, nil
	}
}

/*
Handle error, by responding with message and 500 HTTP code
*/
func handleError(err error) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{Headers: CORSHeaders, Body: err.Error(), StatusCode: 500}, err
}

/*
Main function (entry point)
*/
func main() {
	lambda.Start(handleRequest)
}
