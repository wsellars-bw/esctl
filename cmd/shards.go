package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	shardSort string
	nodeName  string
)

var listShardsCmd = &cobra.Command{
	Use:     "shards",
	Aliases: []string{"shard"},
	Short:   "show information about one or more shard",
	RunE: func(cmd *cobra.Command, args []string) error {
		if nodeName == "" {
			return listShards()
		}
		return listShardsForNode(nodeName)
	},
}

var listShardCountCmd = &cobra.Command{
	Use:   "count",
	Short: "List shard count for each node",
	Long: `
A good rule-of-thumb is to ensure you keep the number of shards per node below 20 per GB heap it 
has configured. A node with a 30GB heap should therefore have a maximum of 600 shards, but the 
further below this limit you can keep it the better. This will generally help the cluster 
stay in good health.

Source: https://www.elastic.co/blog/how-many-shards-should-i-have-in-my-elasticsearch-cluster
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listShardCount()
	},
}

var getShardsCmd = &cobra.Command{
	Use:     "shards",
	Aliases: []string{"shard"},
	Short:   "show information about one or more shard",
}

var getShardAllocationsCmd = &cobra.Command{
	Use:     "allocations",
	Aliases: []string{"alloc"},
	Short:   "Get shard routing allocation",
	RunE: func(cmd *cobra.Command, args []string) error {
		return getShardAllocations()
	},
}

var disableShardCmd = &cobra.Command{
	Use:     "shards",
	Aliases: []string{"shard"},
}

var disableShardAllocationsCmd = &cobra.Command{
	Use:     "allocations",
	Aliases: []string{"alloc"},
	Short:   "Disable shard routing allocations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return disableShardAllocations()
	},
}

var enableShardCmd = &cobra.Command{
	Use:     "shards",
	Aliases: []string{"shard"},
}

var enableShardAllocationsCmd = &cobra.Command{
	Use:     "allocations",
	Aliases: []string{"alloc"},
	Short:   "Enable shard routing allocations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return enableShardAllocations()
	},
}

func getShardAllocations() error {
	b, err := getClusterSettings("**.cluster.routing.allocation.enable")
	if err != nil {
		return err
	}
	settings := map[string]clusterSettings{}
	if err := json.Unmarshal(b, &settings); err != nil {
		return err
	}
	for k, v := range settings {
		if v.Cluster.Allocation.Enable == "" {
			continue
		}
		fmt.Printf("%s\ncluster.routing.allocation.enable: %s\n", k, v.Cluster.Allocation.Enable)
	}
	return nil
}

func disableShardAllocations() error {
	settings := map[string]clusterSettings{"transient": {Cluster{Routing{Allocation{Enable: "none"}}}}}
	b, err := json.Marshal(&settings)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(b)
	resp, err := client.Cluster.PutSettings(buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	fmt.Println(string(b))
	return nil
}

func enableShardAllocations() error {
	settings := map[string]clusterSettings{"transient": {Cluster{Routing{Allocation{Enable: "all"}}}}}
	b, err := json.Marshal(&settings)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(b)
	resp, err := client.Cluster.PutSettings(buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	fmt.Println(string(b))
	return nil
}

func listShards() error {
	resp, err := client.Cat.Shards(client.Cat.Shards.WithHuman(),
		client.Cat.Shards.WithPretty(),
		client.Cat.Shards.WithS(fmt.Sprintf("store:%s,index,shard", shardSort)),
		client.Cat.Shards.WithV(true),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func listShardsJson() error {
	resp, err := client.Cat.Shards(client.Cat.Shards.WithHuman(),
		client.Cat.Shards.WithPretty(),
		client.Cat.Shards.WithS("store:desc,index,shard"),
		client.Cat.Shards.WithV(true),
		client.Cat.Shards.WithFormat("json"),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func listShardsForNode(node string) error {
	resp, err := client.Cat.Shards(client.Cat.Shards.WithHuman(),
		client.Cat.Shards.WithPretty(),
		client.Cat.Shards.WithS(fmt.Sprintf("store:%s,index,shard", shardSort)),
		client.Cat.Shards.WithV(true),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), node) {
			fmt.Println(scanner.Text())
		}
	}
	return nil
}

func listShardCount() error {
	b, err := getNodeStats("indices", "nodes.**.name,nodes.**.indices.shard_stats.total_count")
	if err != nil {
		return err
	}
	nodeStats := &nodeStatsResp{}
	if err := json.Unmarshal(b, nodeStats); err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "name\t shard_count\t")
	for _, n := range nodeStats.Nodes {
		if strings.Contains(n.Name, "data") {
			fmt.Fprintf(w, "%s\t %d\t\n", n.Name, n.IndexStats.Total)
		}
	}
	w.Flush()
	return nil
}

func init() {
	getCmd.AddCommand(getShardsCmd)
	getShardsCmd.AddCommand(getShardAllocationsCmd)
	listCmd.AddCommand(listShardsCmd)
	listShardsCmd.AddCommand(listShardCountCmd)
	listShardsCmd.Flags().StringVarP(&shardSort, "sort", "s", "desc", "sort shard by size. Valid values are asc or desc. Default is desc.")
	listShardsCmd.Flags().StringVar(&nodeName, "node", "", "filter shards based on node name")
	disableCmd.AddCommand(disableShardCmd)
	disableShardCmd.AddCommand(disableShardAllocationsCmd)
	enableCmd.AddCommand(enableShardCmd)
	enableShardCmd.AddCommand(enableShardAllocationsCmd)
}
