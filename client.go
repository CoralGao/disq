package disq

import (
   "net"
   "log"
   "fmt"
   "strings"
   "bufio"
   "os"
   "strconv"
   "sync"
   // "time"
)

type CollectorInterface interface {
   ProcessResult(qid int, result string)
}

type NodeStub struct {
   addr  string
   conn  net.Conn
}

type Client struct {
   nodes          []NodeStub
   collector      CollectorInterface
   config_file    string
}

func NewClient(config_file string) *Client {
   c := new(Client)
   c.config_file = config_file
   return c
}

func (c *Client) Start(index_file, query_file string, collector CollectorInterface) {
   c.collector = collector

   // Connect and distribute queries
   c.connect(index_file)
   go func(qfile string) {
      c.send_queries(qfile)
   }(query_file)

   // Collect and process results
   results := make(chan string)
   c.collect_results(results)

   for r := range(results) {
      items := strings.SplitN(r, " ", 2)
      qid, err := strconv.Atoi(items[0])
      if err != nil {
         log.Fatalln("Missing query id", items)
      }
      res := items[1]
      c.collector.ProcessResult(qid, res)
   }
}

func (c *Client) collect_results(results chan string) {
   var wg sync.WaitGroup

   wg.Add(len(c.nodes))

   go func() {
      wg.Wait()
      close(results)
   }()

   for _, node := range(c.nodes) {
      go func(conn net.Conn) {
         defer wg.Done()
         scanner := bufio.NewScanner(conn)
         defer conn.Close()
         for scanner.Scan() {
            results <- scanner.Text()
         }
      }(node.conn)
   }
}

func (c *Client) connect(index_file string) {
   no_connection := true
   addresses := ReadClientConfig(c.config_file)
   for _, addr := range(addresses) {
      conn, err := net.Dial("tcp", addr)
      if err == nil {
         c.nodes = append(c.nodes, NodeStub{addr, conn})
         fmt.Fprintf(conn, "handshake %s\n", index_file)
         log.Println("connect to", addr)
         no_connection = false
      }
   }
   if no_connection {
      log.Fatalln("Cannot connect to any node.")
   }
}

func (c *Client) send_queries(query_file string) {
   file, e := os.Open(query_file)
   if e != nil {
      log.Fatalln("Unable to open file", query_file)
   }
   defer file.Close()

   scanner := bufio.NewScanner(file)
   for stop,count:=false,0; !stop; {
      for _, node := range(c.nodes) {
         if scanner.Scan() {
            query := scanner.Text()
            fmt.Fprintf(node.conn,"query %d %s\n",count,query)
            count++
            // fmt.Printf("query %d %s\n",count,query)
            // time.Sleep(2 * time.Second)
         } else {
            stop = true
            break
         }
      }
   }
   for _, node := range(c.nodes) {
      fmt.Fprintf(node.conn, "done\n")
   }
}



