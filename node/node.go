package node

import "github.com/remeh/sizedwaitgroup"

type Node struct {
	Collector   string    `help:"collector address"`
	Terraform   Terraform `embed:"" prefix:"terraform."`
	Parallel    int       `help:"Number of processes to run in parallel" default:"1"`
	Directories []string  `arg:"" help:"Directories in which to run terraform"`
}

type Terraform struct {
	PlanCommand string `help:"Command to run in the terraform directory to produce the plan" default:"terraform plan -o plan.bin"`
	PlanFile    string `help:"Name of the created plan file" default:"plan.bin"`
}

func (*Node) Run() error {

	wg := sizedwaitgroup.New(1)
}
