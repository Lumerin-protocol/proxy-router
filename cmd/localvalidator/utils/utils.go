package utils

import "strings"
import "strconv"
import "encoding/json"

//take in a string and return split to inclue the nonce and hash being submitted
func ParseIncomingHash(message string) (uint, []byte) {
	res := strings.Split(message, "|")
	nonce, _ := strconv.ParseUint(res[0], 10, 64)
	hash := res[1]
	return uint(nonce), []byte(hash)

}

//converts a varying byte array with a fixed byte array
func ConvertArray(input []byte) [32]byte {
	newArray := [32]byte{}
	for i := range input {
		newArray[i] = input[i]
	}
	return newArray
}

func BigEndianToLittleEndian(input string) string {
	//convert a big endian to a little endian hex representation
	return ""
}

//returns a mapping of the input message
//assumes that the input message is of JSON format
//creation message needs BH, HashRate, Limit, Difficulty
func CreateMessageMap(message string) map[string]string {
	var myMessage map[string]string
	json.Unmarshal([]byte(message), &myMessage)
	return myMessage
}

func ConvertStringToUint(m string) uint {
	res, _ := strconv.ParseUint(m, 10, 256)
	return uint(res)
}
