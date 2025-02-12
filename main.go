package main

import (
	"embed"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"text/template"

	"github.com/charmbracelet/huh"
	"github.com/tredoe/osutil/user/crypt"
	"github.com/tredoe/osutil/user/crypt/common"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
)

//go:embed autoinstall.yaml.tpl
var f embed.FS

var basePackages = []string{"curl", "gpg", "wget"}

var hypervPackages = []string{"linux-azure",
	"linux-image-azure",
	"linux-headers-azure",
	"linux-tools-common",
	"linux-cloud-tools-common",
	"linux-tools-azure",
	"linux-cloud-tools-azure",
}
var kvmPackages = []string{"spice-vdagent"}

type installation struct {
	RealName     string
	Username     string
	DiskPassword string
	Password     string
	SSHKey       string
	Hostname     string
	Hypervisor   string
	IsVM         bool
	IsHyperV     bool
	IsQemu       bool
	ScriptURL    string
	Packages     []string
}

func main() {

	var c crypt.Crypter
	var s common.Salt
	var shadowHash string
	var saltString string
	var err error

	var ipAddress string
	ip := GetOutboundIP()
	ipAddress = ip.String()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	scriptUrl := fmt.Sprintf("http://%s:%s/intune.sh", ipAddress, port)
	autoinstallUrl := fmt.Sprintf("http://%s:%s/autopilot", ipAddress, port)

	accessible, _ := strconv.ParseBool(os.Getenv("ACCESSIBLE"))

	var diskpasswordTwice string

	var passwordTwice string

	var inst installation
	inst.Packages = basePackages

	c = crypt.New(crypt.SHA512)
	s = sha512_crypt.GetSalt()
	saltString = fmt.Sprintf("%s%s", s.MagicPrefix, saltString)

	form := huh.NewForm(
		huh.NewGroup(huh.NewNote().
			Title("AutoPilot").
			Description("Welcome to _AutoPilot_.\n\nThis application creates and hosts a custom autoinstall.yaml file for Ubuntu.\n\nChoose Next to proceed.\n\n").
			Next(true).
			NextLabel("Next"),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Target Platform").
				Options(
					huh.NewOption("Hyper-V", "hyperv"),
					huh.NewOption("QEMU/KVM", "qemu"),
					huh.NewOption("Bare Metal", "metal"),
				).
				Value(&inst.Hypervisor),
		),
		huh.NewGroup(
			huh.NewInput().
				Value(&inst.RealName).
				Title("Account full name?").
				Placeholder("Margaret Thatcher").
				Description("GECOS User information"),
		),
		huh.NewGroup(
			huh.NewInput().
				Value(&inst.Username).
				Title("Login user name").
				Placeholder("janedoe").
				Description("Linux username"),
			huh.NewInput().
				Value(&inst.Password).
				Title("Password").
				Placeholder("correct-horse-battery-staple").
				EchoMode(huh.EchoModePassword).
				Description("Linux Password"),
			huh.NewInput().
				Value(&passwordTwice).
				Title("Confirm Password").
				Placeholder("correct-horse-battery-staple").
				EchoMode(huh.EchoModePassword).
				Description("Linux Password").
				Validate(func(s string) error {
					if s != inst.Password {
						return errors.New("passwords do not match")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewInput().
				Value(&inst.DiskPassword).
				Title("Disk Encryption Password").
				Placeholder("incorrect-horse-battery-staple").
				Description("LUKS Disk Encryption Password"),
			huh.NewInput().
				Value(&diskpasswordTwice).
				Title("Confirm Disk Encryption Password").
				Placeholder("incorrect-horse-battery-staple").
				Description("LUKS Disk Encryption Password").
				Validate(func(s string) error {
					if s != inst.DiskPassword {
						return errors.New("encryption passwords do not match")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewInput().
				Value(&inst.SSHKey).
				Title("SSH Authorized Key (public)").
				Placeholder("incorrect-horse-battery-staple").
				Description("SSH Public Key"),
		),
		huh.NewGroup(
			huh.NewInput().
				Value(&inst.Hostname).
				Title("Hostname").
				Placeholder("chapterhouse").
				Description("VM Hostname"),
		),
	).WithAccessible(accessible)

	err = form.Run()
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	if inst.Hypervisor == "hyperv" {
		inst.IsHyperV = true
		inst.IsVM = true
		inst.Packages = append(inst.Packages, hypervPackages...)
	}
	if inst.Hypervisor == "qemu" {
		inst.IsQemu = true
		inst.IsVM = true
		inst.Packages = append(inst.Packages, kvmPackages...)
	}
	if inst.Hypervisor == "metal" {
		inst.IsVM = false
	}
	inst.ScriptURL = scriptUrl

	shadowHash, err = c.Generate([]byte(inst.Password), []byte(saltString))
	if err != nil {
		log.Fatal(err)
	}
	inst.Password = shadowHash

	fmt.Println("Autoinstall URL:", autoinstallUrl)
	fmt.Println("Intune Script:", scriptUrl)
	http.HandleFunc("/intune.sh", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(intunescript))
	})

	http.HandleFunc("/autopilot", func(w http.ResponseWriter, r *http.Request) {
		var tmplFile = "autoinstall.yaml.tpl"
		tmpl, err := template.New(tmplFile).ParseFS(f, tmplFile)
		if err != nil {
			panic(err)
		}
		err = tmpl.Execute(w, inst)
		if err != nil {
			log.Fatalf("template executing error: %s", err)

		}

	})

	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil))
}

var intunescript = `#!/bin/bash
echo "hello"
if ! [ $(id -u) = 0 ]; then
   echo "The script need to be run as root." >&2
   exit 1
fi

echo "*** Updating the system"

apt update

apt-get dist-upgrade

apt install curl
read -p "*** Adding the Microsoft package signing key to the the list of trusted keys (Press Enter To Continue)"

curl -sSl https://packages.microsoft.com/keys/microsoft.asc | sudo tee /etc/apt/trusted.gpg.d/microsoft.asc
apt-get install wget gpg
wget -qO- https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor > packages.microsoft.gpg
install -D -o root -g root -m 644 packages.microsoft.gpg /etc/apt/keyrings/packages.microsoft.gpg
rm -f packages.microsoft.gpg

read -p "*** Adding microsoft package repos (Press Enter)"

curl https://packages.microsoft.com/config/ubuntu/22.04/prod.list | sudo tee /etc/apt/sources.list.d/microsoft-ubuntu-jammy-prod.list
sh -c 'echo "deb [arch=amd64] https://packages.microsoft.com/repos/edge stable main" > /etc/apt/sources.list.d/microsoft-edge-dev.list'
sh -c 'echo "deb [arch=amd64,arm64,armhf signed-by=/etc/apt/keyrings/packages.microsoft.gpg] https://packages.microsoft.com/repos/code stable main" > /etc/apt/sources.list.d/vscode.list'

apt update

read -p "*** Installing Intune (Press Enter)"

sh -c 'echo "deb http://us.archive.ubuntu.com/ubuntu/ jammy main restricted" > /etc/apt/sources.list.d/intune_temp.jammymain.sources.list'

sh -c 'echo "deb http://us.archive.ubuntu.com/ubuntu/ jammy updates restricted" > /etc/apt/sources.list.d/intune_temp.jammyupdates.sources.list'

sh -c 'echo "deb http://security.ubuntu.com/ubuntu jammy-security main restricted" > /etc/apt/sources.list.d/intune_temp.jammysecurity.sources.list'


apt update
apt upgrade

read -p "*** Installing JDK *** (Press Enter)"
apt install openjdk-11-jre

read -p "*** Installing Intune *** (Press Enter)"
apt install intune-portal

rm /etc/apt/sources.list.d/intune_temp.*

systemctl --user daemon-reload
apt update

read -p "*** Installing VSCode *** (Press Enter)"

apt install code
`

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
