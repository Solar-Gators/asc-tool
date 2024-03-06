package dataaccess

import "encoding/json"

func fromJson[T any](_json []byte) (*T, error) {
	var deserializedObject T
	err := json.Unmarshal(_json, &deserializedObject)
	if err != nil {
		return nil, err
	}

	return &deserializedObject, nil
}

func toJson[T any](data T) ([]byte, error) {
	return json.Marshal(data)
}
