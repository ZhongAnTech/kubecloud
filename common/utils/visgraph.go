package utils

import (
	"bytes"
	"fmt"
	"strings"
)

type (
	font struct {
		Size int `json:"size"`
	}
	highlight struct {
		Border     string `json:"border"`
		Background string `json:"background"`
	}
	NodeColor struct {
		Border     string    `json:"border"`
		Background string    `json:"background"`
		Highlight  highlight `json:"highlight"`
	}
	EdgeColor struct {
		Color     string `json:"color"`
		Highlight string `json:"highlight"`
	}
	VisNode struct {
		Id          string    `json:"id"`
		Label       string    `json:"label"`
		Shape       string    `json:"shape"`
		Color       NodeColor `json:"color"`
		Size        int       `json:"size"`
		Font        font      `json:"font"`
		BorderWidth int       `json:"borderWidth"`
	}
	VisEdge struct {
		Id       string            `json:"id"`
		Label    string            `json:"label"`
		From     string            `json:"from"`
		To       string            `json:"to"`
		Arrows   string            `json:"arrows"`
		Color    EdgeColor         `json:"color"`
		LabelMap map[string]string `json:"-"`
		Length   int               `json:"length"`
		Width    int               `json:"width"`
	}

	VisGraph struct {
		Nodes []VisNode `json:"nodes"`
		Edges []VisEdge `json:"edges"`
	}
)

const (
	UNKOWN_NODE           = "unknown"
	USER                  = "user"
	DEFAULT_COLOR         = "#848690"
	HIGHLIGHT_COLOR       = "#621ba4"
	NODE_BACKGROUND_COLOR = "#fff"
	NODE_BORDER_COLOR     = "#848690"
	NODE_SIZE             = 40
	NODE_FONT_SIZE        = 28
	NODE_BORDER_WIDTH     = 2
	NODE_SHAPE            = "ellipse"
	EDGE_LENGTH           = 220
	EDGE_WIDTH            = 2
)

func NewVisGraph() *VisGraph {
	return &VisGraph{}
}

// AddEdge adds a new edge to an existing dynamic graph.
func (g *VisGraph) AddEdge(src, target string, lbls map[string]string) {
	var sn, tn *VisNode
	sn = g.getNode(src)
	if sn == nil {
		sn = g.newNode(src)
		g.Nodes = append(g.Nodes, *sn)
	}
	tn = g.getNode(target)
	if tn == nil {
		tn = g.newNode(target)
		g.Nodes = append(g.Nodes, *tn)
	}
	e := VisEdge{
		Id:     sn.Id + "→" + tn.Id,
		From:   sn.Id,
		To:     tn.Id,
		Arrows: "to",
		Color: EdgeColor{
			Color:     DEFAULT_COLOR,
			Highlight: HIGHLIGHT_COLOR,
		},
		LabelMap: make(map[string]string),
		Length:   EDGE_LENGTH,
		Width:    EDGE_WIDTH,
	}
	index := g.checkEdgeExist(e)
	if index == -1 {
		e.LabelMap = lbls
		g.Edges = append(g.Edges, e)
	} else {
		for nk, nv := range lbls {
			value, exist := g.Edges[index].LabelMap[nk]
			if !exist {
				g.Edges[index].LabelMap[nk] = nv
			} else {
				if nv != value {
					g.Edges[index].LabelMap[nk] = nv
				}
			}
		}
	}
}

func (g *VisGraph) MakeEdgeLabel() {
	for i, e := range g.Edges {
		g.Edges[i].Label = labelStr(e.LabelMap)
	}
}

func (g *VisGraph) checkEdgeExist(e VisEdge) int {
	for i, item := range g.Edges {
		if item.From == e.From &&
			item.To == e.To &&
			item.Arrows == e.Arrows {
			return i
		}
	}

	return -1
}

func (g *VisGraph) getNode(name string) *VisNode {
	for _, node := range g.Nodes {
		if node.Label == name {
			return &node
		}
	}

	return nil
}

func (g *VisGraph) newNode(name string) *VisNode {
	//maxId := 0
	//label := name
	//for _, node := range g.Nodes {
	//	if node.Id > maxId {
	//		maxId = node.Id
	//	}
	//}
	//if name == UNKOWN_NODE {
	//	label = USER
	//}
	return &VisNode{
		Id:    name,
		Label: name,
		Color: NodeColor{
			Border:     NODE_BORDER_COLOR,
			Background: NODE_BACKGROUND_COLOR,
			Highlight: highlight{
				Border:     HIGHLIGHT_COLOR,
				Background: NODE_BACKGROUND_COLOR,
			},
		},
		Shape:       NODE_SHAPE,
		Size:        NODE_SIZE,
		BorderWidth: NODE_BORDER_WIDTH,
		Font: font{
			Size: NODE_FONT_SIZE,
		},
	}
}

func labelStr(m map[string]string) string {
	var labelBuf bytes.Buffer
	for _, v := range m {
		labelBuf.WriteString(fmt.Sprintf("%s\n", v))
	}
	return strings.TrimRight(labelBuf.String(), "\n")
}
