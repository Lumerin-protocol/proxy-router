//go:build wireinject
// +build wireinject

package main

import "gitlab.com/TitanInd/hashrouter/app"

const VERSION = "0.01"

func main() {
	appInstance, err := app.InitApp()
	if err != nil {
		panic(err)
	}

	appInstance.Run()
}
