package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"qrcode-decode/qrcode"
)

var (
	detProto = "/models/detect.prototxt"
	detModel = "/models/detect.caffemodel"
	srProto  = "/models/sr.prototxt"
	srModel  = "/models/sr.caffemodel"
	port     = 80
)

var rootCmd = &cobra.Command{Use: "qrdecode", Short: "QR code decoder (CLI + HTTP)"}

var decodeCmd = &cobra.Command{
	Use:   "decode <image>",
	Short: "Decode QR code from image file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		d := qrcode.NewDecoder(detProto, detModel, srProto, srModel)
		text, err := d.Decode(args[0])
		if err != nil {
			return err
		}
		fmt.Println(text)
		return nil
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server for QR decode",
	RunE: func(cmd *cobra.Command, args []string) error {
		d := qrcode.NewDecoder(detProto, detModel, srProto, srModel)

		http.HandleFunc("/decode", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "only POST allowed"})
				return
			}
			file, _, err := r.FormFile("image")
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing 'image' field"})
				return
			}
			defer file.Close()

			limitBody := http.MaxBytesReader(w, file, 20<<20)
			data := make([]byte, 20<<20)
			n, err := limitBody.Read(data)
			if n == 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "empty file"})
				return
			}
			data = data[:n]

			text, err := d.DecodeFromBytes(data)
			if err != nil {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"content": text})
		})

		addr := fmt.Sprintf(":%d", port)
		log.Printf("QR decode server listening on %s", addr)
		return http.ListenAndServe(addr, nil)
	},
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func init() {
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	rootCmd.AddCommand(decodeCmd, serveCmd)

	decodeCmd.Flags().StringVar(&detProto, "detect-proto", detProto, "path to detect.prototxt")
	decodeCmd.Flags().StringVar(&detModel, "detect-model", detModel, "path to detect.caffemodel")
	decodeCmd.Flags().StringVar(&srProto, "sr-proto", srProto, "path to sr.prototxt")
	decodeCmd.Flags().StringVar(&srModel, "sr-model", srModel, "path to sr.caffemodel")

	serveCmd.Flags().StringVar(&detProto, "detect-proto", detProto, "path to detect.prototxt")
	serveCmd.Flags().StringVar(&detModel, "detect-model", detModel, "path to detect.caffemodel")
	serveCmd.Flags().StringVar(&srProto, "sr-proto", srProto, "path to sr.prototxt")
	serveCmd.Flags().StringVar(&srModel, "sr-model", srModel, "path to sr.caffemodel")
	serveCmd.Flags().IntVar(&port, "port", port, "HTTP server port")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
