package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// --- TRẠM 4.2: HÀM GỬI TIN NHẮN TELEGRAM ---
func sendTelegramAlert(message string) {
    // Tải file .env lên bộ nhớ
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Lỗi tải file .env")
    }

    // Đọc biến môi trường thay vì viết cứng
    botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
    chatID := os.Getenv("TELEGRAM_CHAT_ID")

    escapedMessage := url.QueryEscape(message)
    apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s", 
        botToken, chatID, escapedMessage)
    
    http.Get(apiURL)
    fmt.Println("Đã gửi tin nhắn bằng biến môi trường!")
}
// --- TRẠM 1 & 2: CÁC HÀM XỬ LÝ TOKEN (CHÌA KHÓA) ---

// Đọc token từ file token.json
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Lưu token mới xuống file token.json
func saveToken(path string, token *oauth2.Token) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Không thể tạo file lưu token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// Lấy token mới từ Web nếu chưa có file
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Mở link này để lấy mã xác thực: \n%v\n", authURL)
	fmt.Print("Dán mã vào đây và ấn Enter: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Lỗi đọc mã xác thực: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Lỗi đổi mã lấy token: %v", err)
	}
	return tok
}

// Hàm khởi tạo Client có gắn sẵn chìa khóa
func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// --- HÀM CHÍNH (RÁP NỐI TẤT CẢ LẠI) ---
func main() {
	ctx := context.Background()

	// 1. Đọc credentials
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Không thể đọc file credentials.json: %v", err)
	}

	// 2. Cấu hình quyền truy cập (Dùng DriveReadonlyScope để xem được dung lượng)
	config, err := google.ConfigFromJSON(b, drive.DriveReadonlyScope)
	if err != nil {
		log.Fatalf("Lỗi cấu hình Google: %v", err)
	}
	client := getClient(config)

	// 3. Khởi tạo nhân viên Drive Service
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Không thể khởi tạo Drive service: %v", err)
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("BOT GOOGLE DRIVE ALERT ĐANG CHẠY TỰ ĐỘNG...")
	fmt.Println("-------------------------------------------")

	// --- TRẠM 4.4: TỰ ĐỘNG HÓA VỚI TICKER ---
	// Thiết lập kiểm tra mỗi 1 tiếng (Thay bằng 10 * time.Second nếu muốn test nhanh)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		// 4.3: Truy vấn dung lượng
		about, err := srv.About.Get().Fields("storageQuota").Do()
		if err != nil {
			fmt.Printf("Lỗi lấy dữ liệu Drive: %v\n", err)
		} else {
			limit := about.StorageQuota.Limit
			usage := about.StorageQuota.Usage
			usagePercent := (float64(usage) / float64(limit)) * 100

			currentTime := time.Now().Format("15:04:05")
			fmt.Printf("[%s] Kiểm tra dung lượng: %.2f%%\n", currentTime, usagePercent)

			// 4.4: Logic Cảnh báo
			if usagePercent >= 0 {
				msg := fmt.Sprintf("⚠️ CẢNH BÁO: Drive của mày sắp đầy!\nĐã dùng: %.2f%%\nTổng dung lượng: 15GB", usagePercent)
				sendTelegramAlert(msg)
			}
		}

		// Đợi đến nhịp tiếp theo của đồng hồ
		<-ticker.C
	}
}