package main

import (
	"fmt"
	"github.com/phips4/discord-update-patcher/patcher"
	"os"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Println("discord-update-patcher does not take any arguments to start. just run it, the program will ask you if it needs anything.")
		return
	}
	discord := patcher.Discord{}
	discord.SelectDefaultPath()
	discord.SelectLatestVersion()
	discord.SelectModulesPath(discord.Version)

	discord.CreateBackup()

	discord.DeleteModules() //maybe ensure the download is working before deleting
	discord.DownloadFiles()

	modules := discord.InstallModules()
	discord.UpdateJson(modules)

	fmt.Println("discord was successfully updated.")

	//TODO:
	// revert start argument to delete all files and rename backup directory
	// chek if discord is running before starting whole procedure
}
