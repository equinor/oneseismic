package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/namsral/flag"
	"github.com/pebbe/zmq4"
)

type opts struct {
	storageURL   string
	bind string
}

func parseopts() (opts, error) {
	type option struct {
		param *string
		flag  string
		help  string
	}

	opts := opts {}
	params := []option {
		option {
			param: &opts.storageURL,
			flag: "storage-url",
			help: "Storage URL",
		},
		option {
			param: &opts.bind,
			flag: "bind",
			help: "Bind URL e.g. tcp://*:port",
		},
	}

	for _, opt := range params {
		flag.StringVar(opt.param, opt.flag, "", opt.help)
	}
	flag.Parse()
	for _, opt := range params {
		if *opt.param == "" {
			return opts, fmt.Errorf("%s not set", opt.flag)
		}
	}

	return opts, nil
}

type Msg struct {
	pid     string
	n       int
	m       int
	token   string
	payload []byte
}

func parse(msg [][]byte) (Msg, error) {
	p := Msg {}
	if len(msg) != 4 {
		return p, fmt.Errorf("len(msg) = %d; want 4", len(msg))
	}
	p.pid = string(msg[0])
	_, err := fmt.Sscanf(string(msg[1]), "%d/%d", &p.n, &p.m)
	if err != nil {
		return p, fmt.Errorf("%s: %s", err.Error(), string(msg[2]))
	}
	p.token = string(msg[2])
	p.payload = msg[3]
	return p, nil
}

func makeQueue(addr string) (*zmq4.Socket, error) {
	queue, err := zmq4.NewSocket(zmq4.PULL)
	if err != nil {
		log.Fatalf("Unable to create socket: %v", err)
		return nil, err
	}
	err = queue.Bind(addr)
	if err != nil {
		log.Fatalf("Failed to bind %s: %v", addr, err)
		return nil, err
	}

	return queue, nil
}

func listenOnQueue(queue *zmq4.Socket, msgs chan Msg) {
	for {
		m, err := queue.RecvMessageBytes(0)
		if err != nil {
			log.Printf("Unable to get message from queue: %v", err)
			continue
		}

		msg, err := parse(m)
		if err != nil {
			log.Printf("Malformed message: %v", err)
			continue
		}

		msgs <- msg
	}
}

func listen(queue *zmq4.Socket) (chan Msg, error) {
	msgs := make(chan Msg)
	go listenOnQueue(queue, msgs)
	return msgs, nil
}

func run(storageurl string, msgs chan Msg) {
	ctx := context.Background()
	URL, _ := url.Parse(fmt.Sprintf("%s/results", storageurl))

	for {
		msg := <-msgs

		credentials := azblob.NewTokenCredential(msg.token, nil)
		pipeline    := azblob.NewPipeline(credentials, azblob.PipelineOptions{})
		container   := azblob.NewContainerURL(*URL, pipeline)

		/*
		 * Upload in a goroutine to be ready to listen for more messages
		 */
		go func() {
			// TODO: container should already exist, but for now just create it
			// every time
			_, _ = container.Create(
				ctx,
				azblob.Metadata{},
				azblob.PublicAccessNone,
			)

			blobURL := container.NewBlockBlobURL(
				fmt.Sprintf("%s-%d-%d", msg.pid, msg.n, msg.m),
			)
			log.Printf("Uploading %s to azure", blobURL.String())
			_, err := azblob.UploadBufferToBlockBlob(
				ctx,
				msg.payload,
				blobURL,
				azblob.UploadToBlockBlobOptions{},
			)

			if err != nil {
				log.Printf("%s upload failed: %v", msg.pid, err)
			}
		}()
	}
}

/*
 * This entire program is quite primitive in order to explore the overall
 * design. Effectively it's a scheduler for uploading (partial) results to
 * azure, rather than having that complexity alongside downloading and slicing.
 *
 * It's pretty much a pipe. It might actually be a reasonably good idea to have
 * this as a separate program, or at least be balanced with a queue, for maybe
 * load balancing or even a customization point for aggregation.
 */
func main() {
	opts, err := parseopts()
	if err != nil {
		log.Fatalf("Invalid arguments: %v", err)
		os.Exit(1)
	}

	queue, err := makeQueue(opts.bind)
	if err != nil {
		os.Exit(1)
	}

	msgs, err := listen(queue)
	if err != nil {
		os.Exit(1)
	}
	run(opts.storageURL, msgs)
}
