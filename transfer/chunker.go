package transfer

const (
	// DefaultChunkSize is 512KB, the maximum and most efficient chunk size for Telegram.
	DefaultChunkSize = 512 * 1024

	// BigFileThreshold is 10MB; files larger than 10MB must use saveBigFilePart.
	BigFileThreshold = 10 * 1024 * 1024
)

// ProgressCallback reports byte transfer progression to the caller.
type ProgressCallback func(transferred int64, total int64)

// CalculateParts determines the chunk size and total part count for a file of size n.
func CalculateParts(totalSize int64, customChunkSize int) (chunkSize int, totalParts int) {
	chunkSize = DefaultChunkSize
	if customChunkSize > 0 && customChunkSize%1024 == 0 && (512*1024)%customChunkSize == 0 {
		chunkSize = customChunkSize
	}

	if totalSize <= 0 {
		return chunkSize, 1
	}

	parts := int((totalSize + int64(chunkSize) - 1) / int64(chunkSize))
	return chunkSize, parts
}
