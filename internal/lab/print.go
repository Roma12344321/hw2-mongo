package lab

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Log(label string, value any) {
	fmt.Println("\n" + label)
	if value == nil {
		return
	}

	data, err := bson.MarshalExtJSONIndent(value, false, false, "", "  ")
	if err != nil {
		fmt.Println("<ошибка вывода:", err, ">")

		return
	}

	fmt.Println(string(data))
}

func Logf(format string, args ...any) {
	fmt.Printf("\n"+format+"\n", args...)
}

func insertErrorCode(err error) (code int64, details any) {
	if err == nil {
		return 0, nil
	}

	var writeExValue mongo.WriteException
	if errors.As(err, &writeExValue) {
		return writeErrorCode(writeExValue)
	}

	var writeExPtr *mongo.WriteException
	if errors.As(err, &writeExPtr) && writeExPtr != nil {
		return writeErrorCode(*writeExPtr)
	}

	var cmdErr *mongo.CommandError
	if errors.As(err, &cmdErr) {
		return int64(cmdErr.Code), nil
	}

	Log("UNRECOGNIZED INSERT ERROR", bson.D{
		{Key: "type", Value: fmt.Sprintf("%T", err)},
		{Key: "error", Value: err.Error()},
	})

	return -1, err.Error()
}

func writeErrorCode(exception mongo.WriteException) (int64, any) {
	if len(exception.WriteErrors) == 0 {
		return -1, exception.Error()
	}

	writeErr := exception.WriteErrors[0]

	return int64(writeErr.Code), writeErr.Details
}
