package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	pb "github.com/hashicorp/waypoint/pkg/server/gen"
)

// Pipeline represents a single pipeline definition
type Pipeline struct {
	Id   string `hcl:",label"`
	Name string `hcl:"name,optional"`

	StepRaw []*hclStage `hcl:"step,block"`

	Body hcl.Body `hcl:",body"`

	ctx    *hcl.EvalContext
	config *Config
}

type hclPipeline struct {
	Id   string `hcl:",label"`
	Name string `hcl:"name,optional"`

	// We need these raw values to determine the plugins need to be used.
	StepRaw []*hclStage `hcl:"step,block"`

	Body   hcl.Body `hcl:",body"`
	Remain hcl.Body `hcl:",remain"`
}

// Pipelines returns the id of all the defined pipelines
func (c *Config) Pipelines() []string {
	var result []string
	for _, p := range c.hclConfig.Pipelines {
		result = append(result, p.Id)
	}

	return result
}

// Pipeline returns the configured pipeline named n. If the pipeline doesn't
// exist, this will return (nil, nil).
func (c *Config) Pipeline(id string, ctx *hcl.EvalContext) (*Pipeline, error) {
	ctx = appendContext(c.ctx, ctx)

	// Find the pipeline by progressively decoding
	var rawPipeline *hclPipeline
	for _, p := range c.hclConfig.Pipelines {
		if p.Id == id {
			rawPipeline = p
			break
		}
	}
	if rawPipeline == nil {
		return nil, nil
	}

	// Full decode
	var pipeline Pipeline
	if diag := gohcl.DecodeBody(rawPipeline.Body, finalizeContext(ctx), &pipeline); diag.HasErrors() {
		return nil, diag
	}
	pipeline.Id = rawPipeline.Id
	pipeline.Name = rawPipeline.Name
	pipeline.ctx = ctx
	pipeline.config = c
	if pipeline.config != nil {
		pipeline.config.ctx = ctx
	}

	return &pipeline, nil
}

// Ref returns the ref for this pipeline.
func (c *Pipeline) Ref() *pb.Ref_Pipeline {
	return &pb.Ref_Pipeline{
		Ref: &pb.Ref_Pipeline_Id{
			Id: &pb.Ref_PipelineId{
				Id: c.Id,
			},
		},
	}
}

// Step loads the associated section of the configuration
func (c *Pipeline) Step(ctx *hcl.EvalContext) ([]*Step, error) {
	ctx = appendContext(c.ctx, ctx)

	var steps []*Step
	for _, stepRaw := range c.StepRaw {
		body := stepRaw.Body
		scope, err := scopeMatchStage(ctx, stepRaw.WorkspaceScoped, stepRaw.LabelScoped)
		if err != nil {
			return nil, err
		}
		if scope != nil {
			body = scope.Body
		}

		var s Step
		if diag := gohcl.DecodeBody(body, finalizeContext(ctx), &s); diag.HasErrors() {
			return nil, diag
		}
		s.ctx = ctx

		steps = append(steps, &s)
	}

	return steps, nil
}

// StepUse returns the plugin "use" value for a single step
// TODO(briancain): We will leave this unimplemented for now  until we start
// actually executing Step plugins. We will execute each step on their own,
// so we'll probably need a single Step to return the plugin Use label.
func (c *Pipeline) StepUse(ctx *hcl.EvalContext) (string, error) {
	return "", nil
	/*
		if c.StepRaw == nil {
			return "", nil
		}

		useType := c.StepRaw.Use.Type
		stage, err := scopeMatchStage(ctx, c.StepRaw.WorkspaceScoped, c.StepRaw.LabelScoped)
		if err != nil {
			return "", err
		}
		if stage != nil {
			useType = stage.Use.Type
		}

		return useType, nil
	*/
}

// StepsUse iterates over all defined steps in the eval context and returns
// every step plugin type.
// NOTE(briancain): We could gather all of the use plugin labels this way and
// return them as a string?
// Nothing uses this right now, but is included for now.
func (c *Pipeline) StepsUse(ctx *hcl.EvalContext) ([]string, error) {
	if len(c.StepRaw) == 0 {
		return nil, nil
	}

	var useTypes []string

	for _, stepRaw := range c.StepRaw {
		useType := stepRaw.Use.Type
		stage, err := scopeMatchStage(ctx, stepRaw.WorkspaceScoped, stepRaw.LabelScoped)
		if err != nil {
			return nil, err
		}
		if stage != nil {
			useType = stage.Use.Type
		}
		useTypes = append(useTypes, useType)
	}

	return useTypes, nil
}

// StepLabels returns the labels for this stage.
// TODO: see the todo in StepUse for single step verus Many step
func (c *Pipeline) StepLabels(ctx *hcl.EvalContext) ([]map[string]string, error) {
	/*
		if c.StepRaw == nil {
			return nil, nil
		}

		ctx = appendContext(c.ctx, ctx)
		return labels(ctx, c.StepRaw.Body)
	*/
	if len(c.StepRaw) == 0 {
		return nil, nil
	}

	var result []map[string]string

	ctx = appendContext(c.ctx, ctx)
	for _, stepRaw := range c.StepRaw {
		l, err := labels(ctx, stepRaw.Body)
		if err != nil {
			return nil, err
		}

		result = append(result, l)
	}

	return result, nil
}