package db

type InventoryType string

const (
	//InventoryNone       InventoryType = "none"
	InventoryStatic     InventoryType = "static"
	InventoryStaticYaml InventoryType = "static-yaml"
	// InventoryFile means that it is path to the Ansible inventory file
	InventoryFile               InventoryType = "file"
	InventoryTerraformWorkspace InventoryType = "terraform-workspace"
	InventoryTofuWorkspace      InventoryType = "tofu-workspace"
)

// Inventory is the model of an ansible inventory file
type Inventory struct {
	ID        int    `db:"id" json:"id" backup:"-"`
	Name      string `db:"name" json:"name" binding:"required"`
	ProjectID int    `db:"project_id" json:"project_id" backup:"-"`
	Inventory string `db:"inventory" json:"inventory"`

	// accesses hosts in inventory
	SSHKeyID *int      `db:"ssh_key_id" json:"ssh_key_id" backup:"-"`
	SSHKey   AccessKey `db:"-" json:"-" backup:"-"`

	BecomeKeyID *int      `db:"become_key_id" json:"become_key_id" backup:"-"`
	BecomeKey   AccessKey `db:"-" json:"-" backup:"-"`

	// static/file
	Type InventoryType `db:"type" json:"type"`

	// TemplateID is an ID of template which holds the inventory
	// It is not used now but can be used in feature for
	// inventories which can not be used more than one template
	// at once.
	TemplateID *int `db:"template_id" json:"template_id" backup:"-"`

	// RepositoryID is an ID of repo where inventory stored.
	// If null than inventory will be got from template repository.
	RepositoryID *int        `db:"repository_id" json:"repository_id" backup:"-"`
	Repository   *Repository `db:"-" json:"-" backup:"-"`
}

func (e Inventory) GetFilename() string {
	if e.Type != InventoryFile {
		return ""
	}

	return e.Inventory

	//return strings.TrimPrefix(e.Inventory, "/")
}

func FillInventory(d Store, inventory *Inventory) (err error) {
	if inventory.SSHKeyID != nil {
		inventory.SSHKey, err = d.GetAccessKey(inventory.ProjectID, *inventory.SSHKeyID)
	}

	if err != nil {
		return
	}

	if inventory.BecomeKeyID != nil {
		inventory.BecomeKey, err = d.GetAccessKey(inventory.ProjectID, *inventory.BecomeKeyID)
	}

	if err != nil {
		return
	}

	if inventory.RepositoryID != nil {
		var repo Repository
		repo, err = d.GetRepository(inventory.ProjectID, *inventory.RepositoryID)
		if err != nil {
			return
		}

		err = repo.SSHKey.DeserializeSecret()
		if err != nil {
			return
		}

		inventory.Repository = &repo
	}

	return
}
