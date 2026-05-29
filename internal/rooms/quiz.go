package rooms

import (
	"fmt"
	"math/rand"
	"strings"

	"ssh-art/internal/ansi"
)

// QuizCard is a single OSCP drill question with two choices.
type QuizCard struct {
	ID          int
	Question    string
	Choices     [2]string
	Correct     int // 0 or 1
	Explanation string
	Category    string
}

// responsiveInnerWidth computes the usable text width for the quiz modal.
func responsiveInnerWidth(termW int) int {
	innerW := 56
	if termW < 64 {
		innerW = termW - 8
		if innerW < 28 {
			innerW = 28
		}
	} else if termW > 110 {
		innerW = 72
	} else if termW > 90 {
		innerW = 64
	}
	return innerW
}

// DrawQuizModal renders a centered quiz card onto the canvas using exact coordinates.
// The modal is drawn on top of existing content (overwrites cells in its area).
func DrawQuizModal(c *ansi.Canvas, card *QuizCard, revealed bool, chosen int) {
	if card == nil || c.Width < 28 || c.Height < 10 {
		return
	}

	innerW := responsiveInnerWidth(c.Width)

	// Wrap content
	questionLines := ansi.WrapText(card.Question, innerW)
	var choiceLines []string
	var explainLines []string

	if revealed {
		choiceMax := innerW - 6 // "[A] " + " ✓"
		for i, choice := range card.Choices {
			mark := " "
			if i == chosen {
				if i == card.Correct {
					mark = "✓"
				} else {
					mark = "✗"
				}
			} else if i == card.Correct {
				mark = "✓"
			}
			trunc := ansi.TruncateDisplay(choice, choiceMax)
			prefix := fmt.Sprintf("[%s] %s", string(rune('A'+i)), trunc)
			choiceLines = append(choiceLines, prefix+" "+mark)
		}
		explainLines = ansi.WrapText(card.Explanation, innerW)
	} else {
		choiceMax := innerW - 4 // "[A] "
		for i, choice := range card.Choices {
			trunc := ansi.TruncateDisplay(choice, choiceMax)
			prefix := fmt.Sprintf("[%s] %s", string(rune('A'+i)), trunc)
			choiceLines = append(choiceLines, prefix)
		}
	}

	// Compute total height: borders + question + blank + choices + optional explanation
	height := 2 + len(questionLines) + 1 + len(choiceLines)
	if revealed && len(explainLines) > 0 {
		height += 1 + len(explainLines)
	}
	width := innerW + 4 // 2 borders + 2 padding spaces

	// Clamp height to terminal
	if height > c.Height-2 {
		height = c.Height - 2 // leave 1 row margin top and bottom
		if height < 5 {
			height = 5
		}
	}

	// Center horizontally, position near top vertically
	x, _ := ansi.CenterRect(c.Width, c.Height, width, height)
	y := 4 // fixed top margin; card starts around row 4-5
	if y+height > c.Height-1 {
		y = c.Height - height - 1
	}
	if y < 1 {
		y = 1
	}

	// Build and draw the box
	box := ansi.Box{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
		Title:  fmt.Sprintf("OSCP #%02d", card.ID),
	}
	box.Draw(c, ansi.ColorWhite)

	// Draw content inside the box
	row := 0
	for _, line := range questionLines {
		if row >= box.InnerHeight() {
			break
		}
		box.SetContent(c, row, line, ansi.ColorWhite)
		row++
	}

	if row < box.InnerHeight() {
		row++ // blank separator line
	}

	for _, line := range choiceLines {
		if row >= box.InnerHeight() {
			break
		}
		var fg ansi.Color = ansi.ColorWhite
		if revealed {
			if strings.Contains(line, "✓") {
				fg = ansi.MatrixBright
			} else if strings.Contains(line, "✗") {
				fg = ansi.ArcaneRed
			}
		}
		box.SetContent(c, row, line, fg)
		row++
	}

	if revealed && len(explainLines) > 0 {
		if row < box.InnerHeight() {
			row++ // blank separator
		}
		for _, line := range explainLines {
			if row >= box.InnerHeight() {
				break
			}
			box.SetContent(c, row, line, ansi.StarDim)
			row++
		}
	}
}

// MasterQuizPool holds 100 OSCP drill cards.
var MasterQuizPool = []QuizCard{
	// Nmap & Scanning
	{ID: 1, Question: "What nmap flag performs a TCP SYN stealth scan?", Choices: [2]string{"-sS", "-sT"}, Correct: 0, Explanation: "-sS is the default SYN stealth scan. -sT is a full TCP connect scan.", Category: "scanning"},
	{ID: 2, Question: "Which nmap flag enables OS detection?", Choices: [2]string{"-O", "-A"}, Correct: 0, Explanation: "-O enables OS detection. -A enables OS detection, version detection, script scanning, and traceroute.", Category: "scanning"},
	{ID: 3, Question: "What does the nmap flag -sV do?", Choices: [2]string{"Probe open ports for service/version info", "Scan only the top 100 ports"}, Correct: 0, Explanation: "-sV probes open ports to determine service/version information.", Category: "scanning"},
	{ID: 4, Question: "Which nmap script category scans for vulnerabilities?", Choices: [2]string{"vuln", "exploit"}, Correct: 0, Explanation: "The 'vuln' category contains scripts that check for known vulnerabilities.", Category: "scanning"},
	{ID: 5, Question: "What flag performs a UDP scan in nmap?", Choices: [2]string{"-sU", "-sP"}, Correct: 0, Explanation: "-sU performs a UDP scan. -sP was the old ping scan flag (now -sn).", Category: "scanning"},
	{ID: 6, Question: "Which nmap flag skips host discovery and treats all hosts as online?", Choices: [2]string{"-Pn", "-sP"}, Correct: 0, Explanation: "-Pn skips host discovery, scanning all targets as if they are online.", Category: "scanning"},
	{ID: 7, Question: "What does nmap's --top-ports 100 flag do?", Choices: [2]string{"Scans the 100 most common ports", "Scans ports 1-100"}, Correct: 0, Explanation: "--top-ports 100 scans the 100 most common ports based on nmap-services.", Category: "scanning"},
	{ID: 8, Question: "Which flag shows packet trace output in nmap for debugging?", Choices: [2]string{"--packet-trace", "--script-trace"}, Correct: 0, Explanation: "--packet-trace shows all packets sent and received. --script-trace traces NSE scripts.", Category: "scanning"},
	{ID: 9, Question: "What nmap flag performs an ACK scan useful for mapping firewall rulesets?", Choices: [2]string{"-sA", "-sF"}, Correct: 0, Explanation: "-sA is an ACK scan used to map firewall rulesets. -sF is a FIN scan.", Category: "scanning"},
	{ID: 10, Question: "Which nmap output format is greppable?", Choices: [2]string{"-oG", "-oX"}, Correct: 0, Explanation: "-oG is greppable output. -oX is XML output.", Category: "scanning"},

	// Enumeration: SMB
	{ID: 11, Question: "Which tool enumerates SMB shares without credentials?", Choices: [2]string{"enum4linux", "smbclient"}, Correct: 0, Explanation: "enum4linux automates enumeration of SMB shares, users, and OS info. smbclient requires a share name.", Category: "enumeration"},
	{ID: 12, Question: "What is the default port for SMB over NetBIOS?", Choices: [2]string{"139", "445"}, Correct: 0, Explanation: "Port 139 is SMB over NetBIOS. Port 445 is SMB directly over TCP.", Category: "enumeration"},
	{ID: 13, Question: "Which smbclient flag lists shares on a target?", Choices: [2]string{"-L", "-l"}, Correct: 0, Explanation: "smbclient -L lists available shares. -l sets the log basename.", Category: "enumeration"},
	{ID: 14, Question: "What RPC endpoint mapper port does Windows use?", Choices: [2]string{"135", "445"}, Correct: 0, Explanation: "Port 135 is the RPC endpoint mapper (EPM). Port 445 is SMB.", Category: "enumeration"},
	{ID: 15, Question: "Which tool queries RPC for user information over port 135?", Choices: [2]string{"rpcclient", "crackmapexec"}, Correct: 0, Explanation: "rpcclient is a low-level RPC tool for user enumeration and more. crackmapexec is higher-level.", Category: "enumeration"},

	// Enumeration: Web
	{ID: 16, Question: "What HTTP status code indicates a resource is forbidden?", Choices: [2]string{"403", "401"}, Correct: 0, Explanation: "403 Forbidden means the server understood but refuses access. 401 is Unauthorized.", Category: "web"},
	{ID: 17, Question: "Which Gobuster mode searches for directories?", Choices: [2]string{"dir", "dns"}, Correct: 0, Explanation: "gobuster dir performs directory/file brute-forcing. gobuster dns performs DNS subdomain enumeration.", Category: "web"},
	{ID: 18, Question: "What does the curl -I flag retrieve?", Choices: [2]string{"Headers only", "Page content with headers"}, Correct: 0, Explanation: "curl -I fetches headers only using a HEAD request.", Category: "web"},
	{ID: 19, Question: "Which HTTP method is typically used to upload files?", Choices: [2]string{"POST", "PUT"}, Correct: 0, Explanation: "POST is commonly used for file uploads via forms. PUT is for updating/replacing resources.", Category: "web"},
	{ID: 20, Question: "What does a 302 status code indicate?", Choices: [2]string{"Temporary redirect", "Permanent redirect"}, Correct: 0, Explanation: "302 is a temporary redirect. 301 is permanent.", Category: "web"},

	// SQL Injection
	{ID: 21, Question: "Which SQLi payload tests for a boolean-based blind condition?", Choices: [2]string{"' OR '1'='1", "' AND 1=1--"}, Correct: 1, Explanation: "' AND 1=1-- tests if the query still returns true. ' OR '1'='1 is a classic tautology for UNION/in-band.", Category: "web"},
	{ID: 22, Question: "What SQL keyword is used to combine results from multiple SELECT statements?", Choices: [2]string{"UNION", "JOIN"}, Correct: 0, Explanation: "UNION combines results from multiple SELECTs. JOIN merges tables horizontally.", Category: "web"},
	{ID: 23, Question: "Which SQL comment syntax works in MySQL but not standard SQL?", Choices: [2]string{"#", "--"}, Correct: 0, Explanation: "# is a MySQL-specific comment. -- is standard but requires a space after in some DBs.", Category: "web"},
	{ID: 24, Question: "What is the MySQL function to read files from disk?", Choices: [2]string{"LOAD_FILE()", "READ_FILE()"}, Correct: 0, Explanation: "LOAD_FILE() reads files in MySQL. READ_FILE() does not exist.", Category: "web"},
	{ID: 25, Question: "Which SQLMap flag dumps the entire database?", Choices: [2]string{"--dump", "--dbs"}, Correct: 0, Explanation: "--dump dumps table entries. --dbs lists available databases.", Category: "web"},

	// XSS & Web Attacks
	{ID: 26, Question: "Which XSS type stores the payload on the server?", Choices: [2]string{"Stored XSS", "Reflected XSS"}, Correct: 0, Explanation: "Stored XSS persists the payload in the server (database, comments). Reflected requires a crafted URL.", Category: "web"},
	{ID: 27, Question: "What payload tests for basic reflected XSS?", Choices: [2]string{"<script>alert(1)</script>", "<img src=x onerror=alert(1)>"}, Correct: 0, Explanation: "The simple script tag is the most basic XSS test. The img tag is useful for filter evasion.", Category: "web"},
	{ID: 28, Question: "Which file inclusion vulnerability reads local files?", Choices: [2]string{"LFI", "RFI"}, Correct: 0, Explanation: "LFI (Local File Inclusion) reads local files. RFI (Remote) includes remote URLs.", Category: "web"},
	{ID: 29, Question: "What PHP wrapper is used to read files via LFI?", Choices: [2]string{"php://filter", "php://input"}, Correct: 0, Explanation: "php://filter/convert.base64-encode/resource=... reads and encodes files. php://input reads raw POST data.", Category: "web"},
	{ID: 30, Question: "Which header can be manipulated for Host header attacks?", Choices: [2]string{"Host", "Referer"}, Correct: 0, Explanation: "The Host header is used for virtual host routing and can be manipulated for password reset poisoning.", Category: "web"},

	// Linux Privilege Escalation
	{ID: 31, Question: "Which command lists files with SUID bit set?", Choices: [2]string{"find / -perm -4000", "find / -perm -2000"}, Correct: 0, Explanation: "-perm -4000 finds SUID files. -2000 finds SGID files.", Category: "linux-privesc"},
	{ID: 32, Question: "What does the SUID bit allow?", Choices: [2]string{"Execute with file owner's privileges", "Execute with group's privileges"}, Correct: 0, Explanation: "SUID executes with the file owner's privileges. SGID uses the group's privileges.", Category: "linux-privesc"},
	{ID: 33, Question: "Which file lists cron jobs for all users?", Choices: [2]string{"/etc/crontab", "/var/spool/cron/crontabs"}, Correct: 0, Explanation: "/etc/crontab is the system-wide cron file. /var/spool/cron/crontabs is per-user.", Category: "linux-privesc"},
	{ID: 34, Question: "What command shows processes running as root?", Choices: [2]string{"ps aux | grep root", "ps -ef | grep root"}, Correct: 0, Explanation: "Both work, but ps aux is the BSD-style format commonly taught in OSCP.", Category: "linux-privesc"},
	{ID: 35, Question: "Which tool automates Linux privilege escalation checks?", Choices: [2]string{"linPEAS", "winPEAS"}, Correct: 0, Explanation: "linPEAS is for Linux. winPEAS is for Windows.", Category: "linux-privesc"},
	{ID: 36, Question: "What capability allows a binary to bypass DAC permissions?", Choices: [2]string{"CAP_SETUID", "CAP_NET_BIND_SERVICE"}, Correct: 0, Explanation: "CAP_SETUID allows changing UID, effectively bypassing DAC. CAP_NET_BIND_SERVICE binds to privileged ports.", Category: "linux-privesc"},
	{ID: 37, Question: "Which command shows capabilities on a binary?", Choices: [2]string{"getcap", "setcap"}, Correct: 0, Explanation: "getcap displays capabilities. setcap sets them.", Category: "linux-privesc"},
	{ID: 38, Question: "What wildcard character in a tar command can lead to privilege escalation?", Choices: [2]string{"*", "?"}, Correct: 0, Explanation: "A wildcard * in a tar command can be exploited via crafted filenames like --checkpoint-action=exec=sh.", Category: "linux-privesc"},
	{ID: 39, Question: "Which sudo misconfiguration allows shell escape?", Choices: [2]string{"vim, less, man", "cat, ls, echo"}, Correct: 0, Explanation: "vim, less, and man have shell escape features (:!, !bash). cat/ls/echo do not.", Category: "linux-privesc"},
	{ID: 40, Question: "What file contains user sudo privileges?", Choices: [2]string{"/etc/sudoers", "/etc/sudoers.d/"}, Correct: 0, Explanation: "/etc/sudoers is the main file. /etc/sudoers.d/ contains included files.", Category: "linux-privesc"},
	{ID: 41, Question: "Which kernel exploit technique leverages eBPF?", Choices: [2]string{"eBPF verifier bugs", "Dirty COW"}, Correct: 0, Explanation: "eBPF verifier bugs are a modern kernel exploit class. Dirty COW is a race condition in copy-on-write.", Category: "linux-privesc"},
	{ID: 42, Question: "What does a writable /etc/passwd allow?", Choices: [2]string{"Adding a root user", "Changing kernel parameters"}, Correct: 0, Explanation: "Writable /etc/passwd allows adding a user with UID 0 (root).", Category: "linux-privesc"},
	{ID: 43, Question: "Which command shows PATH environment variable?", Choices: [2]string{"echo $PATH", "env PATH"}, Correct: 0, Explanation: "echo $PATH displays the PATH. env PATH shows PATH only if set as a command.", Category: "linux-privesc"},
	{ID: 44, Question: "What is the sticky bit used for on directories?", Choices: [2]string{"Only owner can delete files", "Everyone can execute files"}, Correct: 0, Explanation: "Sticky bit (/tmp) prevents users from deleting others' files.", Category: "linux-privesc"},
	{ID: 45, Question: "Which file lists mounted filesystems?", Choices: [2]string{"/etc/fstab", "/proc/mounts"}, Correct: 0, Explanation: "/etc/fstab defines mount points. /proc/mounts shows currently mounted filesystems.", Category: "linux-privesc"},

	// Windows Privilege Escalation
	{ID: 46, Question: "Which Windows service configuration can be exploited for privesc?", Choices: [2]string{"SERVICE_CHANGE_CONFIG", "SERVICE_QUERY_STATUS"}, Correct: 0, Explanation: "SERVICE_CHANGE_CONFIG allows modifying service binaries for privilege escalation.", Category: "windows-privesc"},
	{ID: 47, Question: "What tool enumerates Windows privileges and vulnerabilities?", Choices: [2]string{"winPEAS", "linPEAS"}, Correct: 0, Explanation: "winPEAS is the Windows Privilege Escalation Awesome Script.", Category: "windows-privesc"},
	{ID: 48, Question: "Which Windows privilege allows impersonation tokens?", Choices: [2]string{"SeImpersonatePrivilege", "SeBackupPrivilege"}, Correct: 0, Explanation: "SeImpersonatePrivilege allows token impersonation (Potato attacks). SeBackupPrivilege bypasses ACLs.", Category: "windows-privesc"},
	{ID: 49, Question: "What is the Windows equivalent of /etc/passwd?", Choices: [2]string{"SAM database", "NTDS.dit"}, Correct: 0, Explanation: "SAM stores local user accounts. NTDS.dit is the Active Directory database.", Category: "windows-privesc"},
	{ID: 50, Question: "Which tool extracts passwords from Windows memory?", Choices: [2]string{"mimikatz", "hashcat"}, Correct: 0, Explanation: "mimikatz extracts plaintext credentials from LSASS. hashcat cracks hashes offline.", Category: "windows-privesc"},
	{ID: 51, Question: "What registry key stores autologon credentials?", Choices: [2]string{"HKLM\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\Winlogon", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run"}, Correct: 0, Explanation: "The Winlogon key stores DefaultUserName and DefaultPassword for autologon.", Category: "windows-privesc"},
	{ID: 52, Question: "Which Windows scheduled task folder is user-writable?", Choices: [2]string{"C:\\Windows\\Tasks", "C:\\Windows\\System32\\Tasks"}, Correct: 0, Explanation: "C:\\Windows\\Tasks is historically writable. System32\\Tasks requires admin.", Category: "windows-privesc"},
	{ID: 53, Question: "What does AlwaysInstallElevated allow?", Choices: [2]string{"MSI installation as SYSTEM", "Auto-elevation of all executables"}, Correct: 0, Explanation: "AlwaysInstallElevated in registry allows MSI packages to install with SYSTEM privileges.", Category: "windows-privesc"},
	{ID: 54, Question: "Which PowerShell command lists services with unquoted paths?", Choices: [2]string{"gwmi win32_service | ?{$_.PathName -notmatch '\"'} ", "Get-Service | Where-Object {$_.Status -eq 'Running'}"}, Correct: 0, Explanation: "The first filters for services lacking quotes in their path, enabling path hijacking.", Category: "windows-privesc"},
	{ID: 55, Question: "What attack exploits unquoted service paths with spaces?", Choices: [2]string{"Path hijacking", "DLL hijacking"}, Correct: 0, Explanation: "Unquoted service paths with spaces allow placing malicious executables in the path.", Category: "windows-privesc"},
	{ID: 56, Question: "Which Windows tool schedules tasks from command line?", Choices: [2]string{"schtasks", "at"}, Correct: 0, Explanation: "schtasks is the modern task scheduler. at is deprecated.", Category: "windows-privesc"},
	{ID: 57, Question: "What is the default RID of the Administrator account?", Choices: [2]string{"500", "501"}, Correct: 0, Explanation: "RID 500 is the built-in Administrator. 501 is the Guest account.", Category: "windows-privesc"},
	{ID: 58, Question: "Which Windows feature stores credentials for SSO?", Choices: [2]string{"Credential Manager", "Windows Vault"}, Correct: 0, Explanation: "Credential Manager stores passwords and certificates for SSO. Windows Vault is a subset.", Category: "windows-privesc"},
	{ID: 59, Question: "What tool enumerates Windows hotfixes?", Choices: [2]string{"wmic qfe get Caption,Description,HotFixID,InstalledOn", "systeminfo"}, Correct: 0, Explanation: "wmic qfe lists installed hotfixes. systeminfo also shows them but with more noise.", Category: "windows-privesc"},
	{ID: 60, Question: "Which token privilege enables bypassing traverse checking?", Choices: [2]string{"SeChangeNotifyPrivilege", "SeDebugPrivilege"}, Correct: 0, Explanation: "SeChangeNotifyPrivilege (BypassTraverseChecking) is given to all users by default.", Category: "windows-privesc"},

	// Buffer Overflow
	{ID: 61, Question: "Which register holds the return address on x86?", Choices: [2]string{"EIP", "ESP"}, Correct: 0, Explanation: "EIP is the instruction pointer (return address). ESP is the stack pointer.", Category: "buffer-overflow"},
	{ID: 62, Question: "What does EIP stand for?", Choices: [2]string{"Extended Instruction Pointer", "Execution Input Pointer"}, Correct: 0, Explanation: "EIP = Extended Instruction Pointer on x86. RIP on x64.", Category: "buffer-overflow"},
	{ID: 63, Question: "Which Python struct format packs a 32-bit little-endian integer?", Choices: [2]string{"<I", ">I"}, Correct: 0, Explanation: "<I is little-endian unsigned int. >I is big-endian.", Category: "buffer-overflow"},
	{ID: 64, Question: "What is the purpose of a NOP sled?", Choices: [2]string{"Increase probability of hitting shellcode", "Decrypt the payload"}, Correct: 0, Explanation: "A NOP sled increases the chance that a jump lands in the shellcode region.", Category: "buffer-overflow"},
	{ID: 65, Question: "Which mona.py command finds a JMP ESP instruction?", Choices: [2]string{"!mona jmp -r esp", "!mona find -type instr -s 'jmp esp'"}, Correct: 0, Explanation: "!mona jmp -r esp finds JMP ESP gadgets. The second is valid but less common syntax.", Category: "buffer-overflow"},
	{ID: 66, Question: "What does ASLR protect against?", Choices: [2]string{"Predictable memory addresses", "Stack-based overflows"}, Correct: 0, Explanation: "ASLR randomizes memory addresses. DEP/NX prevents execution from the stack.", Category: "buffer-overflow"},
	{ID: 67, Question: "Which technique bypasses ASLR when a module has no ASLR?", Choices: [2]string{"Return to a non-ASLR module", "ROP chain"}, Correct: 0, Explanation: "Returning to a non-ASLR module provides predictable addresses. ROP chains reuse code gadgets.", Category: "buffer-overflow"},
	{ID: 68, Question: "What does DEP/NX prevent?", Choices: [2]string{"Executing code from the stack/heap", "Writing to read-only memory"}, Correct: 0, Explanation: "DEP/NX marks stack/heap as non-executable. It does not prevent writing.", Category: "buffer-overflow"},
	{ID: 69, Question: "Which Immunity Debugger command sets a breakpoint?", Choices: [2]string{"F2", "F9"}, Correct: 0, Explanation: "F2 sets a breakpoint. F9 runs the program.", Category: "buffer-overflow"},
	{ID: 70, Question: "What pattern does mona generate for finding the crash offset?", Choices: [2]string{"Cyclic pattern", "De Bruijn sequence"}, Correct: 0, Explanation: "mona generates a cyclic pattern. A De Bruijn sequence is the mathematical concept behind it.", Category: "buffer-overflow"},

	// Password Attacks & Hashing
	{ID: 71, Question: "Which hash type does Windows use for local accounts?", Choices: [2]string{"NTLM", "NTLMv2"}, Correct: 0, Explanation: "Windows stores NTLM hashes (MD4 of UTF-16 password). NTLMv2 is the authentication protocol.", Category: "passwords"},
	{ID: 72, Question: "What tool cracks password hashes using wordlists?", Choices: [2]string{"hashcat", "John the Ripper"}, Correct: 0, Explanation: "Both crack hashes, but hashcat is GPU-accelerated and the standard for speed.", Category: "passwords"},
	{ID: 73, Question: "Which hashcat mode cracks NTLM hashes?", Choices: [2]string{"-m 1000", "-m 0"}, Correct: 0, Explanation: "-m 1000 is NTLM. -m 0 is MD5.", Category: "passwords"},
	{ID: 74, Question: "What does the -O flag do in hashcat?", Choices: [2]string{"Optimized kernel", "Output to file"}, Correct: 0, Explanation: "-O uses optimized kernels for faster cracking at the cost of password length. -o outputs to file.", Category: "passwords"},
	{ID: 75, Question: "Which tool generates password lists based on target information?", Choices: [2]string{"cewl", "crunch"}, Correct: 0, Explanation: "cewl spiders a website to generate wordlists. crunch generates all combinations.", Category: "passwords"},
	{ID: 76, Question: "What is the hash mode for Net-NTLMv2 in hashcat?", Choices: [2]string{"-m 5600", "-m 5500"}, Correct: 0, Explanation: "-m 5600 is Net-NTLMv2. -m 5500 is Net-NTLMv1.", Category: "passwords"},
	{ID: 77, Question: "Which attack tries every possible password combination?", Choices: [2]string{"Brute force", "Dictionary attack"}, Correct: 0, Explanation: "Brute force tries all combinations. Dictionary uses a wordlist.", Category: "passwords"},
	{ID: 78, Question: "What does Hydra's -L flag specify?", Choices: [2]string{"Username wordlist", "Password wordlist"}, Correct: 0, Explanation: "-L is the login/username wordlist. -P is the password wordlist.", Category: "passwords"},

	// Tunneling & Pivoting
	{ID: 79, Question: "Which SSH flag creates a local port forward?", Choices: [2]string{"-L", "-R"}, Correct: 0, Explanation: "-L forwards a local port to a remote destination. -R creates a remote forward.", Category: "pivoting"},
	{ID: 80, Question: "What does ssh -R 8080:localhost:80 do?", Choices: [2]string{"Remote forward: remote 8080 -> local 80", "Local forward: local 8080 -> remote 80"}, Correct: 0, Explanation: "-R creates a remote forward, exposing local port 80 on the remote host's port 8080.", Category: "pivoting"},
	{ID: 81, Question: "Which tool creates a SOCKS proxy through SSH?", Choices: [2]string{"ssh -D", "ssh -L"}, Correct: 0, Explanation: "ssh -D creates a dynamic SOCKS proxy. ssh -L is static port forwarding.", Category: "pivoting"},
	{ID: 82, Question: "What is the default SOCKS port for proxychains?", Choices: [2]string{"1080", "3128"}, Correct: 0, Explanation: "1080 is the default SOCKS port. 3128 is a common Squid HTTP proxy port.", Category: "pivoting"},
	{ID: 83, Question: "Which Meterpreter command routes traffic through a session?", Choices: [2]string{"route add", "portfwd add"}, Correct: 0, Explanation: "route add routes subnet traffic through a session. portfwd adds a single port forward.", Category: "pivoting"},
	{ID: 84, Question: "What tool tunnels traffic over DNS queries?", Choices: [2]string{"iodine", "dnscat2"}, Correct: 0, Explanation: "iodine tunnels IP over DNS. dnscat2 is a C2 over DNS tool.", Category: "pivoting"},
	{ID: 85, Question: "Which chisel command creates a reverse SOCKS proxy?", Choices: [2]string{"chisel client <server> R:socks", "chisel server --reverse"}, Correct: 0, Explanation: "chisel client with R:socks creates a reverse SOCKS tunnel from server to client.", Category: "pivoting"},

	// Metasploit
	{ID: 86, Question: "Which Meterpreter command spawns a system shell?", Choices: [2]string{"shell", "execute"}, Correct: 0, Explanation: "shell spawns an interactive system shell. execute runs a single command.", Category: "metasploit"},
	{ID: 87, Question: "What command lists available payloads in msfconsole?", Choices: [2]string{"show payloads", "show options"}, Correct: 0, Explanation: "show payloads lists payloads. show options shows configurable options.", Category: "metasploit"},
	{ID: 88, Question: "Which payload type connects back to the attacker?", Choices: [2]string{"reverse", "bind"}, Correct: 0, Explanation: "reverse payloads connect back. bind payloads listen on the target.", Category: "metasploit"},
	{ID: 89, Question: "What Meterpreter command migrates to another process?", Choices: [2]string{"migrate", "pivot"}, Correct: 0, Explanation: "migrate moves the Meterpreter session to another process ID.", Category: "metasploit"},
	{ID: 90, Question: "Which msfvenom flag specifies the output format?", Choices: [2]string{"-f", "-p"}, Correct: 0, Explanation: "-f sets the output format (exe, elf, raw). -p selects the payload.", Category: "metasploit"},

	// Enumeration Methodology
	{ID: 91, Question: "What is the first step in the penetration testing methodology?", Choices: [2]string{"Information Gathering", "Exploitation"}, Correct: 0, Explanation: "Information gathering (reconnaissance) always comes before exploitation.", Category: "methodology"},
	{ID: 92, Question: "Which phase follows vulnerability analysis?", Choices: [2]string{"Exploitation", "Post-exploitation"}, Correct: 0, Explanation: "After finding vulnerabilities, you exploit them. Post-exploitation comes after successful exploitation.", Category: "methodology"},
	{ID: 93, Question: "What is the purpose of the post-exploitation phase?", Choices: [2]string{"Maintain access and gather evidence", "Scan for vulnerabilities"}, Correct: 0, Explanation: "Post-exploitation focuses on persistence, data exfiltration, and lateral movement.", Category: "methodology"},
	{ID: 94, Question: "Which document is delivered at the end of a pentest?", Choices: [2]string{"Penetration Test Report", "Scope Agreement"}, Correct: 0, Explanation: "The penetration test report documents findings, risks, and remediation. Scope agreement is signed before testing.", Category: "methodology"},
	{ID: 95, Question: "What does the PTES stand for?", Choices: [2]string{"Penetration Testing Execution Standard", "Penetration Test Evaluation System"}, Correct: 0, Explanation: "PTES is the Penetration Testing Execution Standard, a comprehensive testing methodology.", Category: "methodology"},

	// Networking
	{ID: 96, Question: "Which ICMP type is a ping request?", Choices: [2]string{"Type 8", "Type 0"}, Correct: 0, Explanation: "ICMP Type 8 is Echo Request (ping). Type 0 is Echo Reply.", Category: "networking"},
	{ID: 97, Question: "What is the ARP protocol used for?", Choices: [2]string{"Resolving IP to MAC addresses", "Routing between networks"}, Correct: 0, Explanation: "ARP resolves IP addresses to MAC addresses on the local network.", Category: "networking"},
	{ID: 98, Question: "Which port does DNS use by default?", Choices: [2]string{"53", "43"}, Correct: 0, Explanation: "DNS uses UDP/TCP port 53. Port 43 is WHOIS.", Category: "networking"},
	{ID: 99, Question: "What does TTL stand for in networking?", Choices: [2]string{"Time To Live", "Total Transfer Limit"}, Correct: 0, Explanation: "TTL (Time To Live) limits how long a packet can exist on the network.", Category: "networking"},
	{ID: 100, Question: "Which protocol is used for secure remote administration?", Choices: [2]string{"SSH", "Telnet"}, Correct: 0, Explanation: "SSH provides encrypted remote administration. Telnet sends data in plaintext.", Category: "networking"},
}

// PickRandomQuiz returns a random quiz card from the master pool.
func PickRandomQuiz() *QuizCard {
	if len(MasterQuizPool) == 0 {
		return nil
	}
	c := MasterQuizPool[rand.Intn(len(MasterQuizPool))]
	return &c
}
