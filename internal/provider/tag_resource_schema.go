package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func tagValuesAttribute() schema.ListAttribute {
	return schema.ListAttribute{
		MarkdownDescription: "List of values for this tag.",
		Required:            true,
		ElementType:         types.StringType,
	}
}

func tagsMapAttribute(objectName string) schema.MapAttribute {
	return schema.MapAttribute{
		MarkdownDescription: "All tags to set on the " + objectName + ". This map is authoritative: every apply writes " +
			"exactly these tags, so any tag not listed here is removed. Only these keys are tracked on refresh — " +
			"tags added under other keys outside Terraform do not show up as drift.",
		ElementType: types.ListType{ElemType: types.StringType},
		Required:    true,
	}
}

func trackedTagValues(tags map[string][]string, key string, tracked []string) (values []string, found bool) {
	if values, found = tags[key]; found {
		return values, true
	}
	// The API drops a tag with no values, so a tracked empty list stays present. On import tracked is nil.
	if tracked != nil && len(tracked) == 0 {
		return []string{}, true
	}
	return nil, false
}
