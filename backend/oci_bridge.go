package main

import (
	"context"
	"errors"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
)

func GetOCIProviderByID(accountID int) (common.ConfigurationProvider, error) {
	var tenancyID, userID, fingerprint, region, encryptedKey string
	err := DB.QueryRow("SELECT tenancy_id, user_id, fingerprint, region, encrypted_key FROM oci_accounts WHERE id = ?", accountID).Scan(&tenancyID, &userID, &fingerprint, &region, &encryptedKey)
	if err != nil {
		return nil, errors.New("未找到指定的甲骨文云账户")
	}

	privateKeyPlainText, err := DecryptText(encryptedKey)
	if err != nil {
		return nil, errors.New("解密私钥失败，请检查 MASTER_KEY 完整性")
	}

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

func TestOCILink(accountID int) (string, error) {
	provider, err := GetOCIProviderByID(accountID)
	if err != nil {
		return "", err
	}

	client, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		return "", err
	}

	// 修复点：规范获取租户 ID 并转化为指针类型
	tID, err := provider.TenancyOCID()
	if err != nil {
		return "", err
	}

	req := identity.GetTenancyRequest{TenancyId: common.String(tID)}
	resp, err := client.GetTenancy(context.Background(), req)
	if err != nil {
		return "", err
	}

	return *resp.Tenancy.Name, nil
}
