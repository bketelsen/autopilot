#cloud-config
autoinstall:
    version: 1
    locale: "en_US.UTF-8"
    storage:
      layout:
        name: lvm
        password: '{{ .DiskPassword }}'
    identity:
        realname: '{{ .RealName }}'
        hostname: {{ .Hostname }}
        username: {{ .Username }}
        password: '{{ .Password }}'
    ssh:
        install-server: yes
        authorized-keys:
          - {{ .SSHKey }}
        allow-pw: no
    packages:
{{range .Packages}}      - {{.}}
{{end}}
    late-commands:
      - curtin in-target -- sed -i 's/retry=3/retry=3 dcredit=-1 ocredit=-1 ucredit=-1 lcredit=-1 minlen=12/' /etc/pam.d/common-password
      - curtin in-target -- apt update
      - curtin in-target -- apt-get install -y curl
      - curtin in-target -- curl -o /root/intune.sh '{{ .ScriptURL }}'
      - curtin in-target -- chmod +x /root/intune.sh



