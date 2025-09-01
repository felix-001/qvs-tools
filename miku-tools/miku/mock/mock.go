package mock

import (
	"encoding/json"
	"fmt"
	"net/http"

	commonModel "github.com/qbox/mikud-live/common/model"
)

func MockLived() {

	http.HandleFunc("/api/v1/streamregister", func(w http.ResponseWriter, r *http.Request) {
		// Handle the request for /api/v1/streamregister
		//fmt.Fprintf(w, "Stream register endpoint")
		resp := commonModel.StreamPublishResponse{
			Uid:       123,
			ErrCode:   "100",
			Message:   "success",
			ConnectId: "123",
			Bucket:    "test",
			Key:       "test",
		}
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			fmt.Printf("Error marshalling response: %v\n", err)
			return
		}
		w.Write(jsonResp)
	})
	fmt.Println("Starting HTTP server on port 9099...")
	err := http.ListenAndServe(":9099", nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
