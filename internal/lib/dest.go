package lib

import (
	"net/url"
	"strings"
)

func SetWorkerName(u *url.URL, workerName string) {
	accountName, _, _ := SplitUsername(u.User.Username())
	pwd, _ := u.User.Password()
	u.User = url.UserPassword(JoinUsername(accountName, workerName), pwd)
}

func SplitUsername(username string) (accountName string, workerName string, ok bool) {
	return strings.Cut(username, ".")
}

func JoinUsername(accountName, userName string) string {
	return accountName + "." + userName
}
