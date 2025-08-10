package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type Client struct {
	Name string
	ID   int
	Conn net.Conn
}

var (
	lockClients    sync.Mutex
	clientList     []Client                 // like vector<pair<string,int>> (name, id)
	groupsToClient = make(map[string][]int) // group -> []clientID
	clientToGroup  = make(map[int]string)   // clientID -> group
	idToClient     = make(map[int]*Client)  // clientID -> ptr
	serverListener net.Listener
	nextClientID   = 1
)

// remove int from slice, preserving order
func removeIntFromSlice(a []int, x int) []int {
	out := a[:0]
	for _, v := range a {
		if v != x {
			out = append(out, v)
		}
	}
	return out
}

func closeClient(clientID int) {
	lockClients.Lock()
	defer lockClients.Unlock()

	c := idToClient[clientID]
	if c != nil && c.Conn != nil {
		_ = c.Conn.Close()
	}

	// remove from clientList
	for i := range clientList {
		if clientList[i].ID == clientID {
			clientList = append(clientList[:i], clientList[i+1:]...)
			break
		}
	}

	// remove from group mappings
	if grp, ok := clientToGroup[clientID]; ok {
		groupsToClient[grp] = removeIntFromSlice(groupsToClient[grp], clientID)
		delete(clientToGroup, clientID)
	}

	delete(idToClient, clientID)
}

func sendTo(clientID int, msg string) error {
	c := idToClient[clientID]
	if c == nil || c.Conn == nil {
		return fmt.Errorf("client missing")
	}
	_, err := c.Conn.Write([]byte(msg))
	return err
}

func joinGroup(clientID int, raw string) int {
	groupName := ""
	if len(raw) >= 6 {
		groupName = strings.TrimSpace(raw[6:])
	}
	lockClients.Lock()
	defer lockClients.Unlock()

	msg := ""
	if _, inGroup := clientToGroup[clientID]; inGroup {
		msg = "You are already a part of a group."
	} else {
		if _, ok := groupsToClient[groupName]; !ok {
			groupsToClient[groupName] = []int{}
			msg = "Created group " + groupName
		} else {
			msg = "Successfully joined group " + groupName
		}
		groupsToClient[groupName] = append(groupsToClient[groupName], clientID)
		clientToGroup[clientID] = groupName
	}

	msg += "\n"
	if err := sendTo(clientID, msg); err != nil {
		closeClient(clientID)
		return -1
	}
	return 1
}

func getUsersList(clientID int) int {
	lockClients.Lock()
	defer lockClients.Unlock()

	var usersList string
	if _, ok := clientToGroup[clientID]; !ok {
		usersList = "Connected Users:"
		for j := 0; j < len(clientList); j++ {
			usersList += "\n" + fmt.Sprintf("%d. %s", j+1, clientList[j].Name)
		}
	} else {
		groupName := clientToGroup[clientID]
		usersList = "Users connected to " + groupName + ":"
		for j := 0; j < len(groupsToClient[groupName]); j++ {
			id := groupsToClient[groupName][j]
			clientName := ""
			for k := 0; k < len(clientList); k++ {
				if clientList[k].ID == id {
					clientName = clientList[k].Name
					break
				}
			}
			usersList += "\n" + fmt.Sprintf("%d. %s", j+1, clientName)
		}
	}
	usersList += "\n"

	if err := sendTo(clientID, usersList); err != nil {
		closeClient(clientID)
		return -1
	}
	return 1
}

func clientRoutine(clientID int) {
	c := idToClient[clientID]
	if c == nil || c.Conn == nil {
		closeClient(clientID)
		return
	}

	ask := "Please enter your username: "
	if _, err := c.Conn.Write([]byte(ask)); err != nil {
		closeClient(clientID)
		return
	}

	// First recv = username chunk (like C++: single recv(), not line-based)
	buf := make([]byte, 1024)
	n, err := c.Conn.Read(buf)
	if err != nil || n <= 0 {
		// mimic perror + close in C++
		closeClient(clientID)
		return
	}
	clientName := string(buf[:n])

	lockClients.Lock()
	c.Name = clientName
	clientList = append(clientList, Client{Name: clientName, ID: clientID, Conn: c.Conn})
	fmt.Println(clientName)
	lockClients.Unlock()

	welcome := "Welcome " + clientName + "! You can use the following commands:\n" +
		"/users - List all connected users\n" +
		"/join <group_name> - Join a group\n" +
		"/groups - List all available groups\n" +
		"/leave - Leave the current group\n"
	if _, err := c.Conn.Write([]byte(welcome)); err != nil {
		closeClient(clientID)
		return
	}

	// Main recv loop: treat each Read() chunk as a message (like C++ recv)
	for {
		n, err := c.Conn.Read(buf)
		if err != nil || n <= 0 {
			// received <=0 : close and exit
			closeClient(clientID)
			return
		}
		temp := string(buf[:n])

		switch {
		case strings.HasPrefix(temp, "/users"):
			if getUsersList(clientID) < 0 {
				return
			}
		case strings.HasPrefix(temp, "/join"):
			if joinGroup(clientID, temp) < 0 {
				return
			}
		case strings.HasPrefix(temp, "/groups"):
			lockClients.Lock()
			groupsList := "Available Groups:"
			for grp, ids := range groupsToClient {
				groupsList += "\n" + grp + " (" + fmt.Sprintf("%d", len(ids)) + " user/s)"
			}
			groupsList += "\n"
			lockClients.Unlock()
			if err := sendTo(clientID, groupsList); err != nil {
				closeClient(clientID)
				return
			}
		case strings.HasPrefix(temp, "/leave"):
			lockClients.Lock()
			if grp, ok := clientToGroup[clientID]; ok {
				groupsToClient[grp] = removeIntFromSlice(groupsToClient[grp], clientID)
				delete(clientToGroup, clientID)
				lockClients.Unlock()

				msg := "You have left the group " + grp + "\n"
				if err := sendTo(clientID, msg); err != nil {
					closeClient(clientID)
					return
				}
			} else {
				lockClients.Unlock()
				msg := "You are not part of any group.\n"
				if err := sendTo(clientID, msg); err != nil {
					closeClient(clientID)
					return
				}
			}
		default:
			// Broadcast: group or global
			lockClients.Lock()
			name := c.Name
			if grp, ok := clientToGroup[clientID]; ok {
				out := "[" + grp + "] " + name + ": " + temp
				recipients := append([]int(nil), groupsToClient[grp]...)
				lockClients.Unlock()

				for _, id := range recipients {
					if id == clientID {
						continue
					}
					if err := sendTo(id, out); err != nil {
						closeClient(id)
					}
				}
			} else {
				out := "[Global] " + name + ": " + temp
				recipients := make([]int, 0, len(clientList))
				for _, meta := range clientList {
					recipients = append(recipients, meta.ID)
				}
				lockClients.Unlock()

				for _, id := range recipients {
					if id == clientID {
						continue
					}
					if err := sendTo(id, out); err != nil {
						closeClient(id)
					}
				}
			}
		}
	}
}

func main() {
	// SIGINT handling (Ctrl-C)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT)
	go func() {
		<-sigc
		fmt.Print("Detected exit")
		if serverListener != nil {
			_ = serverListener.Close()
		}
		lockClients.Lock()
		for _, c := range idToClient {
			if c.Conn != nil {
				_ = c.Conn.Close()
			}
		}
		lockClients.Unlock()
		os.Exit(0)
	}()

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("listen failed:", err)
		return
	}
	serverListener = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			// likely listener closed on SIGINT
			return
		}

		lockClients.Lock()
		myID := nextClientID
		nextClientID++
		idToClient[myID] = &Client{Name: "", ID: myID, Conn: conn}
		lockClients.Unlock()

		go clientRoutine(myID)
	}
}
