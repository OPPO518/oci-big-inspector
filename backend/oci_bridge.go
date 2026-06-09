package main

import (
	"context"
	"errors"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
)

// GetOCIProviderByID 根据数据库中的账户 ID，动态在内存中还原并构建 OCI SDK 凭证提供者
func GetOCIProviderByID(accountID int) (common.ConfigurationProvider, error) {
	var tenancyID, userID, fingerprint, region, encryptedKey string
	
	// 1. 从 SQLite 中检索出该账号的加密配置
	err := DB.QueryRow("SELECT tenancy_id, user_id, fingerprint, region, encrypted_key FROM oci_accounts WHERE id = ?", accountID).Scan(&tenancyID, &userID, &fingerprint, &region, &encryptedKey)
	if err != nil {
		return nil, errors.New("未找到指定的甲骨文云账户")
	}

	// 2. 调用 crypto.go 解密出内存中的明文私钥文本
	privateKeyPlainText, err := DecryptText(encryptedKey)
	if err != nil {
		return nil, errors.New("解密私钥失败，请检查 MASTER_KEY 完整性")
	}

	// 3. 动态建立免落盘官方标准 Provider
	provider := common.NewRawConfigurationProvider(
		tenancyID,
		userID,
		region,
		fingerprint,
		privateKeyPlainText,
		nil,
	)

	return provider, nil
}

// TestOCILink 测试某个账号的 API 连通性 (用于在网页上点“测试连接”按钮)
func TestOCILink(accountID int) (string, error) {
	provider, err := GetOCIProviderByID(accountID)
	if err != nil {
		return "", err
	}

	client, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		return "", err
	}

	// 使用 context.Background() 保证请求的生命周期安全
	req := identity.GetTenancyRequest{TenancyId: provider.TenancyOCID()}
	resp, err := client.GetTenancy(context.Background(), req)
	if err != nil {
		return "", err
	}

	return *resp.Tenancy.Name, nil
}
