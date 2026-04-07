package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"AndersSpringborg/jira-cli/pkg/jira/cloud"
)

// TransitionRequest struct holds request data for issue transition request.
type TransitionRequest struct {
	Update     *TransitionRequestUpdate `json:"update,omitempty"`
	Fields     *TransitionRequestFields `json:"fields,omitempty"`
	Transition *TransitionRequestData   `json:"transition"`
}

// TransitionRequestUpdate struct holds a list of operations to perform on the issue screen field.
type TransitionRequestUpdate struct {
	Comment []struct {
		Add struct {
			Body string `json:"body"`
		} `json:"add"`
	} `json:"comment,omitempty"`
}

// TransitionRequestFields struct holds a list of issue screen fields to update along with sub-fields.
type TransitionRequestFields struct {
	Assignee *struct {
		Name string `json:"name"`
	} `json:"assignee,omitempty"`
	Resolution *struct {
		Name string `json:"name"`
	} `json:"resolution,omitempty"`
}

// TransitionRequestData is a transition request data.
type TransitionRequestData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type transitionResponse struct {
	Expand      string        `json:"expand"`
	Transitions []*Transition `json:"transitions"`
}

// Transitions fetches valid transitions for an issue using the generated cloud client
// GET /issue/{key}/transitions endpoint.
func (c *Client) Transitions(key string) ([]*Transition, error) {
	if c.cloud == nil {
		return nil, fmt.Errorf("cloud client not initialized")
	}

	resp, err := c.cloud.GetTransitionsWithResponse(context.Background(), key, nil)
	if err != nil {
		return nil, err
	}
	if resp.HTTPResponse == nil {
		return nil, ErrEmptyResponse
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, parseCloudError(resp.Body, resp.HTTPResponse)
	}
	if resp.JSON200 == nil {
		return nil, ErrEmptyResponse
	}

	return convertTransitions(resp.JSON200), nil
}

// TransitionsV2 fetches valid transitions for an issue using v2 version of the GET /issue/{key}/transitions endpoint.
func (c *Client) TransitionsV2(key string) ([]*Transition, error) {
	return c.transitions(key, apiVersion2)
}

func (c *Client) transitions(key, ver string) ([]*Transition, error) {
	path := fmt.Sprintf("/issue/%s/transitions", key)

	var (
		res *http.Response
		err error
	)

	switch ver {
	case apiVersion2:
		res, err = c.GetV2(context.Background(), path, nil)
	default:
		res, err = c.Get(context.Background(), path, nil)
	}

	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, formatUnexpectedResponse(res)
	}

	var out transitionResponse

	err = json.NewDecoder(res.Body).Decode(&out)

	return out.Transitions, err
}

// Transition moves issue from one state to another using the generated cloud client
// POST /issue/{key}/transitions endpoint.
func (c *Client) Transition(key string, data *TransitionRequest) (int, error) {
	if c.cloud == nil {
		return 0, fmt.Errorf("cloud client not initialized")
	}

	body := cloud.DoTransitionJSONRequestBody{
		Transition: &cloud.IssueTransition{
			Id: &data.Transition.ID,
		},
	}

	// Build update map for comments if present.
	if data.Update != nil && len(data.Update.Comment) > 0 {
		updateMap := make(map[string][]cloud.FieldUpdateOperation)
		var ops []cloud.FieldUpdateOperation
		for _, c := range data.Update.Comment {
			addMap := map[string]interface{}{"body": c.Add.Body}
			ops = append(ops, cloud.FieldUpdateOperation{Add: &addMap})
		}
		updateMap["comment"] = ops
		body.Update = &updateMap
	}

	// Build fields map for assignee/resolution if present.
	if data.Fields != nil {
		fieldsMap := make(map[string]interface{})
		if data.Fields.Assignee != nil {
			fieldsMap["assignee"] = map[string]string{"name": data.Fields.Assignee.Name}
		}
		if data.Fields.Resolution != nil {
			fieldsMap["resolution"] = map[string]string{"name": data.Fields.Resolution.Name}
		}
		body.Fields = &fieldsMap
	}

	resp, err := c.cloud.DoTransitionWithResponse(context.Background(), key, body)
	if err != nil {
		return 0, err
	}
	if resp.HTTPResponse == nil {
		return 0, ErrEmptyResponse
	}
	if resp.StatusCode() != http.StatusNoContent {
		return resp.StatusCode(), parseCloudError(resp.Body, resp.HTTPResponse)
	}
	return resp.StatusCode(), nil
}

// convertTransitions maps the generated cloud Transitions type to our domain type.
func convertTransitions(t *cloud.Transitions) []*Transition {
	if t.Transitions == nil {
		return nil
	}

	var out []*Transition
	for _, tr := range *t.Transitions {
		trans := &Transition{}
		if tr.Id != nil {
			trans.ID = json.Number(*tr.Id)
		}
		if tr.Name != nil {
			trans.Name = *tr.Name
		}
		if tr.IsAvailable != nil {
			trans.IsAvailable = *tr.IsAvailable
		}
		out = append(out, trans)
	}

	return out
}
