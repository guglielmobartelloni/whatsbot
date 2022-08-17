package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type WrappedClient struct {
	WAClient       *whatsmeow.Client
	eventHandlerID uint32
}

func (wrappedClient *WrappedClient) register() {
	wrappedClient.eventHandlerID = wrappedClient.WAClient.AddEventHandler(wrappedClient.myEventHandler)
}

func (wrappedClient *WrappedClient) myEventHandler(evt interface{}) {
	// Handle event and access mycli.WAClient
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message: ", v.Message.GetConversation())
		fmt.Println("JID: ", v.Info.Sender)

	}
}

func main() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	myClient := &WrappedClient{
		WAClient: client,
	}
	myClient.register()
	connectToWhatsapp(myClient)

	// Takes the recipient from the environment variable RECIPIENT
	jid, err := types.ParseJID(os.Getenv("RECIPIENT"))
	if err != nil {
		panic(err)
	}
	sendMessage(myClient, jid, "Test message")

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Disconnect the client
	myClient.WAClient.Disconnect()
}

func sendMessage(myClient *WrappedClient, jid types.JID, message string) {
	myClient.WAClient.SendMessage(context.Background(), jid, "", &waProto.Message{
		Conversation: proto.String(message),
	})

}

func connectToWhatsapp(myClient *WrappedClient) {

	if myClient.WAClient.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := myClient.WAClient.GetQRChannel(context.Background())
		err := myClient.WAClient.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err := myClient.WAClient.Connect()
		if err != nil {
			panic(err)
		}
	}

}
