package main

// Both instances share the markup and validation while keeping independent IDs,
// fields, photos, geolocation and submission state.
type leadRequestData struct {
	LandingPageData
	Prefix, FormID, AnchorID, Position string
}

func requestFormData(page LandingPageData, position string) leadRequestData {
	data := leadRequestData{LandingPageData: page, FormID: "leadForm", AnchorID: "lead-form", Position: "hero"}
	if position == "bottom" {
		data.Prefix = "bottom-"
		data.FormID = "leadFormBottom"
		data.AnchorID = "lead-form-bottom"
		data.Position = "bottom"
	}
	return data
}
