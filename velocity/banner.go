package velocity

import (
	"fmt"
)

const (
	reset   = "\033[0m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
)

func printBanner(addr string) {
	fmt.Println(cyan + `
██╗░░░██╗███████╗██╗░░░░░░█████╗░░█████╗░██╗████████╗██╗░░░██╗
██║░░░██║██╔════╝██║░░░░░██╔══██╗██╔══██╗██║╚══██╔══╝╚██╗░██╔╝
╚██╗░██╔╝█████╗░░██║░░░░░██║░░██║██║░░╚═╝██║░░░██║░░░░╚████╔╝░
░╚████╔╝░██╔══╝░░██║░░░░░██║░░██║██║░░██╗██║░░░██║░░░░░╚██╔╝░░
░░╚██╔╝░░███████╗███████╗╚█████╔╝╚█████╔╝██║░░░██║░░░░░░██║░░░
░░░╚═╝░░░╚══════╝╚══════╝░╚════╝░░╚════╝░╚═╝░░░╚═╝░░░░░░╚═╝░░░` + reset)

	fmt.Println(magenta + "⚡ Velocity Web Framework – Ultra Fast, Minimal, Fun!" + reset)
	fmt.Println(yellow+"🌐 Listening on : "+reset, addr)
	fmt.Println(blue + "📦 Version      : v0.1.0" + reset)
	fmt.Println(green + "📖 Docs         : https://github.com/juven0/Velocity" + reset)
	fmt.Println(red + "🛑 Press Ctrl+C to stop the server" + reset)
	fmt.Println()
}
