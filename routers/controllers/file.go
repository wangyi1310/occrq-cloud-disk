package controllers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wangyi1310/mycloud-disk/conf"
)

// FileController 文件控制器（已包含Upload方法，此处添加Download方法）
type FileController struct{}

func (c *FileController) Upload(ctx *gin.Context) {
	// 1. 从请求中获取上传文件（表单字段名：file）
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(400, gin.H{
			"code":    400,
			"message": "未获取到文件或文件格式错误",
			"error":   err.Error(),
		})
		return
	}
	defer file.Close() // 确保文件流最终关闭

	// 2. 构建文件保存路径（防止路径遍历攻击）
	filename := header.Filename
	savePath := filepath.Join(conf.SystemConfig.UploadDir, filename)

	// 3. 创建目标文件
	dstFile, err := os.Create(savePath)
	if err != nil {
		ctx.JSON(500, gin.H{
			"code":    500,
			"message": "创建文件失败",
			"error":   err.Error(),
		})
		return
	}
	defer dstFile.Close() // 确保目标文件流关闭

	// 4. 复制文件内容（从上传流到目标文件）
	if _, err := io.Copy(dstFile, file); err != nil {
		ctx.JSON(500, gin.H{
			"code":    500,
			"message": "文件内容保存失败",
			"error":   err.Error(),
		})
		return
	}

	// 5. 返回上传成功响应
	ctx.JSON(200, gin.H{
		"code":     200,
		"message":  "文件上传成功",
		"filename": filename,
		"size":     header.Size, // 文件大小（字节）
		"savePath": savePath,    // 服务器保存路径
	})
}

func (c *FileController) Download(ctx *gin.Context) {
	// 1. 获取URL中的文件名（防止路径遍历攻击）
	filename := ctx.Param("filename")
	if filename == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "文件名不能为空",
		})
		return
	}

	// 2. 构建文件路径（与上传目录保持一致）
	uploadDir := "./uploads"
	filePath := filepath.Join(uploadDir, filename)

	// 3. 验证文件是否存在
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"code":     http.StatusNotFound,
				"message":  "文件不存在",
				"filename": filename,
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "打开文件失败",
				"error":   err.Error(),
			})
		}
		return
	}
	defer file.Close()

	// 4. 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "获取文件信息失败",
			"error":   err.Error(),
		})
		return
	}
	fileSize := fileInfo.Size()

	// 5. 处理Range请求头（断点续传核心逻辑）
	rangeHeader := ctx.GetHeader("Range")
	if rangeHeader == "" {
		// 无Range头：返回完整文件
		ctx.Header("Content-Length", strconv.FormatInt(fileSize, 10))
		ctx.Header("Content-Type", "application/octet-stream")
		ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		ctx.Status(http.StatusOK)
		io.Copy(ctx.Writer, file)
		return
	}

	// 6. 解析Range请求（格式：bytes=start-end）
	start, end, err := parseRange(rangeHeader, fileSize)
	if err != nil {
		ctx.Header("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		ctx.JSON(http.StatusRequestedRangeNotSatisfiable, gin.H{
			"code":    http.StatusRequestedRangeNotSatisfiable,
			"message": "无效的Range请求格式",
			"error":   err.Error(),
		})
		return
	}

	// 7. 验证范围有效性
	if start >= fileSize || end >= fileSize {
		ctx.Header("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		ctx.JSON(http.StatusRequestedRangeNotSatisfiable, gin.H{
			"code":    http.StatusRequestedRangeNotSatisfiable,
			"message": fmt.Sprintf("请求范围超出文件大小（文件大小：%d bytes）", fileSize),
		})
		return
	}

	// 8. 返回部分文件内容（206 Partial Content）
	contentLength := end - start + 1
	ctx.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	ctx.Header("Accept-Ranges", "bytes")
	ctx.Header("Content-Length", strconv.FormatInt(contentLength, 10))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	ctx.Status(http.StatusPartialContent)

	// 定位文件指针并传输指定范围内容
	file.Seek(start, io.SeekStart)
	io.CopyN(ctx.Writer, file, contentLength)
}

// parseRange 解析Range请求头（内部辅助函数）
func parseRange(rangeHeader string, fileSize int64) (start, end int64, err error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, errors.New("仅支持bytes类型范围请求（格式：bytes=start-end）")
	}

	rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return 0, 0, errors.New("范围格式错误，应为bytes=start-end")
	}

	// 解析起始位置
	if parts[0] == "" {
		// 格式：bytes=-end（最后end字节）
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end <= 0 {
			return 0, 0, errors.New("结束位置必须为正整数")
		}
		start = fileSize - end
		end = fileSize - 1
	} else {
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || start < 0 {
			return 0, 0, errors.New("起始位置必须为非负整数")
		}

		// 解析结束位置
		if parts[1] == "" {
			// 格式：bytes=start-（从start到文件末尾）
			end = fileSize - 1
		} else {
			end, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil || end < start {
				return 0, 0, errors.New("结束位置必须大于等于起始位置")
			}
		}
	}

	return start, end, nil
}
