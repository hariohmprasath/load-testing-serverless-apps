package db

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	uuid2 "github.com/google/uuid"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Constants
const (
	TableName     = "SurveyPoll"
	MaxRetry      = 10
	SurveyId      = "SurveyId"
	SurveyName    = "SurveyName"
	QuestionId    = "QuestionId"
	Question      = "Question"
	Ttl           = "ttl"
	DefaultExpiry = 60
)

// Survey
type Survey struct {
	Name                string
	SurveyId            string
	QuestionsAndAnswers []QuestionsAndAnswers
}

// Survey --> 1 to many --> Questions and answers
type QuestionsAndAnswers struct {
	Question   string
	QuestionId int
	Answers    []Answers
}

// Questions --> 1 to many --> Answers
type Answers struct {
	Answer   string
	AnswerId int
	Vote     int
}

var Client = DynamoDbClient()

/*
Method to describe dynamodb table
*/
func DescribeTable() (*string, error) {
	// Describe table
	dResponse, dErr := Client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(TableName),
	})

	if dErr != nil {
		if aErr, ok := dErr.(awserr.Error); ok {
			// Report error anything other than resource not found
			if aErr.Code() != dynamodb.ErrCodeResourceNotFoundException {
				fmt.Println(dErr.Error())
			}
		}
		return nil, dErr
	} else {
		return dResponse.Table.TableStatus, nil
	}
}

/*
Wait till table is completed created or failed
*/
func WaitForCompletion() (bool, error) {
	counter := 1
	tableStatus, err := DescribeTable()
	if err == nil {
		for tableStatus != nil &&
			(*tableStatus == dynamodb.TableStatusCreating || *tableStatus == dynamodb.TableStatusDeleting) &&
			counter < MaxRetry {
			// Wait for 5 seconds and retry
			time.Sleep(5 * time.Second)
			tableStatus, err = DescribeTable()
			counter++
		}

		if counter == MaxRetry {
			return false, nil
		} else {
			return true, nil
		}
	}
	return false, err
}

/*
Vote function which increments the answers_<id>_vote column by 1
*/
func Vote(surveyId string, questionId string, answerId string) error {
	var keyAttributeMap = map[string]*dynamodb.AttributeValue{}
	keyAttributeMap[SurveyId] = &dynamodb.AttributeValue{S: aws.String(surveyId)}
	keyAttributeMap[QuestionId] = &dynamodb.AttributeValue{N: aws.String(questionId)}

	var expressionAttributeMap = map[string]*dynamodb.AttributeValue{}
	expressionAttributeMap[":inc"] = &dynamodb.AttributeValue{N: aws.String("1")}

	// Generate dynamodb column name
	answerVoteId := fmt.Sprintf("answer_%s_vote", answerId)

	// Update item
	_, err := Client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName:                 aws.String(TableName),
		Key:                       keyAttributeMap,
		UpdateExpression:          aws.String(fmt.Sprintf("SET %s = %s + :inc", answerVoteId, answerVoteId)),
		ExpressionAttributeValues: expressionAttributeMap,
	})

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}

/*
Get all questions and answers associated with the survey
*/
func GetSurvey(surveyId string) (*Survey, error) {
	// Run query based on partition key
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v1": {
				S: aws.String(surveyId),
			},
		},
		KeyConditionExpression: aws.String(fmt.Sprintf("%s = :v1", SurveyId)),
		TableName:              aws.String(TableName),
	}

	result, err := Client.Query(input)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	// Map dynamodb response to Survey struct
	survey := Survey{
		SurveyId:            surveyId,
		QuestionsAndAnswers: []QuestionsAndAnswers{},
	}
	for _, item := range result.Items {
		questionId, _ := strconv.Atoi(*item[QuestionId].N)
		survey.Name = *item[SurveyName].S
		qAndA := QuestionsAndAnswers{
			Question:   *item[Question].S,
			QuestionId: questionId,
			Answers:    []Answers{},
		}

		var keyMap = map[int]Answers{}
		var keys []int
		for key, value := range item {
			if strings.HasPrefix(key, "answer") {
				token, _ := strconv.Atoi(strings.Split(key, "_")[1])

				var target = Answers{}
				if val, ok := keyMap[token]; ok {
					target = val
				} else {
					keys = append(keys, token)
				}

				target.AnswerId = token
				if strings.HasSuffix(key, "vote") {
					target.Vote, _ = strconv.Atoi(*value.N)
				} else {
					target.Answer = *value.S
				}
				keyMap[token] = target
			}
		}

		// Sort the answers by key
		sort.Ints(keys)

		// Item and populate the array
		for _, k := range keys {
			qAndA.Answers = append(qAndA.Answers, keyMap[k])
		}

		survey.QuestionsAndAnswers = append(survey.QuestionsAndAnswers, qAndA)
	}

	return &survey, nil
}

/*
Create a survey with name, questions and answers
*/
func CreateSurvey(survey Survey) (*string, error) {

	// Create table if not exists
	_, cErr := CreateTableIfNotExists()
	if cErr != nil {
		return nil, cErr
	}

	// Generate survey ID based on UUID
	uuid := uuid2.New().String()

	var writeRequests []*dynamodb.WriteRequest
	for qCounter, q := range survey.QuestionsAndAnswers {
		var attributeMap = map[string]*dynamodb.AttributeValue{}

		// PK
		attributeMap[SurveyId] = &dynamodb.AttributeValue{S: aws.String(uuid)}
		// Hash
		attributeMap[QuestionId] = &dynamodb.AttributeValue{N: aws.String(strconv.Itoa(qCounter))}
		// Survey Name
		attributeMap[SurveyName] = &dynamodb.AttributeValue{S: aws.String(survey.Name)}
		// Question
		attributeMap[Question] = &dynamodb.AttributeValue{S: aws.String(q.Question)}

		// Answer details
		for aCounter, a := range q.Answers {
			attributeMap[fmt.Sprintf("answer_%d_label", aCounter)] = &dynamodb.AttributeValue{S: aws.String(a.Answer)}
			attributeMap[fmt.Sprintf("answer_%d_vote", aCounter)] = &dynamodb.AttributeValue{N: aws.String("0")}
		}

		// Write request
		writeRequest := &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: attributeMap,
			},
		}

		writeRequests = append(writeRequests, writeRequest)
	}

	// Batch request
	_, err := Client.BatchWriteItem(&dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			TableName: writeRequests,
		},
	})

	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return &uuid, nil
}

/*
Create table if not exists
*/
func CreateTableIfNotExists() (bool, error) {
	_, dErr := DescribeTable()

	// Table doesnt exist so create new
	if dErr != nil {
		input := &dynamodb.CreateTableInput{
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String(SurveyId),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String(QuestionId),
					AttributeType: aws.String("N"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String(SurveyId),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String(QuestionId),
					KeyType:       aws.String("RANGE"),
				},
			},
			BillingMode: aws.String(dynamodb.BillingModePayPerRequest),
			TableName:   aws.String(TableName),
		}

		// Create table
		_, err := Client.CreateTable(input)
		if err != nil {
			fmt.Println(err.Error())
		}

		// Table created wait for completion
		success, wErr := WaitForCompletion()

		if wErr != nil {
			fmt.Println(wErr)
			return false, wErr
		} else if !success {
			return false, nil
		}

		// Enable TTL column
		_, ttlErr := DynamoDbClient().UpdateTimeToLive(&dynamodb.UpdateTimeToLiveInput{
			TableName: aws.String(TableName),
			TimeToLiveSpecification: &dynamodb.TimeToLiveSpecification{
				AttributeName: aws.String(Ttl),
				Enabled:       aws.Bool(true),
			},
		})

		if ttlErr != nil {
			fmt.Println(ttlErr.Error())
			return false, ttlErr
		}

		return true, nil
	} else {
		return true, nil
	}
}

/*
Insert default seed data for survey data
*/
func RebuildData() (*string, error) {
	// Read all data & unmarshal to dynamodb list
	var attributeList []map[string]*dynamodb.AttributeValue

	unmarshalErr := json.Unmarshal([]byte(DynamodbItems), &attributeList)
	if unmarshalErr != nil {
		fmt.Println("Unable to unmarshal the file", unmarshalErr.Error())
		return nil, unmarshalErr
	}

	// Calculate expiry (defaulted to 1 hour)
	expires := time.Now().Add(time.Duration(DefaultExpiry) * time.Minute)
	expiresInt := expires.UnixNano()
	expiresString := strconv.FormatInt(expiresInt, 10)

	// Create write requests
	var writeRequests []*dynamodb.WriteRequest
	var uuid = ""
	for _, attribute := range attributeList {
		if len(uuid) == 0 {
			uuid = *attribute["SurveyId"].S
		}

		// Set TTL
		attribute[Ttl] = &dynamodb.AttributeValue{S: aws.String(expiresString)}

		writeRequest := &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: attribute,
			},
		}

		writeRequests = append(writeRequests, writeRequest)
	}

	// Batch request
	_, batchErr := Client.BatchWriteItem(&dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			TableName: writeRequests,
		},
	})

	if batchErr != nil {
		fmt.Println("Failed while inserting the batch", batchErr.Error())
		return nil, batchErr
	}

	return &uuid, nil
}

/*
Delete table
*/
func DeleteTable() error {
	_, err := DynamoDbClient().DeleteTable(&dynamodb.DeleteTableInput{
		TableName: aws.String(TableName),
	})

	if err != nil {
		if aErr, ok := err.(awserr.Error); ok {
			// Report error anything other than resource not found
			if aErr.Code() != dynamodb.ErrCodeResourceNotFoundException {
				fmt.Println(err.Error())
			}
		} else {
			fmt.Println("Error while deleting table", err.Error())
			return err
		}
	} else {
		_, wErr := WaitForCompletion()

		if wErr != nil {
			fmt.Println("Error while waiting for deletion", wErr.Error())
			return wErr
		}
	}

	return nil
}

// Get Dynamodb to read data
func DynamoDbClient() *dynamodb.DynamoDB {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create Dynamodb client
	return dynamodb.New(sess)
}
