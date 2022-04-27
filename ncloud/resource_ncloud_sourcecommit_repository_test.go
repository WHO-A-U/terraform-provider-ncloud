package ncloud

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/sourcecommit"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceNcloudSourceCommitRepository_basic(t *testing.T) {
	fmt.Println("test basic repository fmt.Println")

	var repository sourcecommit.GetRepositoryDetailResponseResult
	resourceName := "ncloud_sourcecommit_repository.test-repo-basic"
	repositoryName := getTestRepositoryName()
	repositoryDesc := fmt.Sprintf("description of %v", repositoryName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSourceCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceNcloudSourceCommitRepositoryConfig(repositoryName, repositoryDesc),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSourceCommitRepositoryExists(resourceName, &repository),
					resource.TestCheckResourceAttr(resourceName, "name", repositoryName),
					resource.TestCheckResourceAttr(resourceName, "description", repositoryDesc),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccResourceNcloudSourceCommitRepositoryConfig(name string, description string) string {
	return fmt.Sprintf(`
resource "ncloud_sourcecommit_repository" "test-repo-basic" {
	name = "%[1]s"
	description = "%[2]s"
	filesafer = true
}
`, name, description)
}

func testAccCheckSourceCommitRepositoryExists(n string, repository *sourcecommit.GetRepositoryDetailResponseResult) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Repository Id is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)

		resp, err := getRepositoryById(context.Background(), config, rs.Primary.ID)
		if err != nil {
			return err
		}

		*repository = *resp
		return nil
	}
}

func testAccCheckSourceCommitRepositoryDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ncloud_sourcecommit_repository" {
			continue
		}

		repositories, err := getRepositoryList(context.Background(), config)
		if err != nil {
			return err
		}

		for _, repository := range repositories {
			if strconv.Itoa(ncloud.IntValue(repository.Id)) == rs.Primary.ID {
				return fmt.Errorf("Repository still exists")
			}
		}
	}
	return nil
}

func getTestRepositoryName() string {
	rInt := acctest.RandIntRange(1, 9999)
	testRepositoryName := fmt.Sprintf("tf-%d-repository", rInt)
	return testRepositoryName
}
