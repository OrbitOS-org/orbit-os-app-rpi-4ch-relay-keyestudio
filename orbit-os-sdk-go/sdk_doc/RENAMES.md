# SDK Method Renames

This file tracks method and parameter renames across the SDK.
Each entry includes the full instructions so a refactoring agent can apply or verify the change.

Status: ✅ aplicado | ⏳ pendente

---

## UpdateManager

### `InstallOtaFromFile` → `Update` ✅

**File:** `client/update_manager.go`

- Rename `func (u *UpdateManager) InstallOtaFromFile(ctx context.Context, otaPath string) error` → `func (u *UpdateManager) Update(ctx context.Context, orbitPath string) error`
- Update the comment above the method accordingly
- Search for all call sites: `\.InstallOtaFromFile(` → `\.Update(`

**File:** `docs/api-reference.html`

- In the `services` array, find the entry with `id: 'update-ota'`
- Update `name:`, `desc:`, the `sig(...)` signature string and the example code string

---

### param `otaPath` → `orbitPath` ✅

**File:** `client/update_manager.go`

- Replace all occurrences of `otaPath` with `orbitPath` inside `func (u *UpdateManager) Update(...)`

**File:** `docs/api-reference.html`

- In the `Update` method entry, update `params` name and the `sig(...)` signature string

---

## PackageManager

### `GetInstalledPackages` → `ListInstalledPackages` ✅

**File:** `client/package_manager.go`

- Rename `func (p *PackageManager) GetInstalledPackages()` → `func (p *PackageManager) ListInstalledPackages()`
- Update the comment above the method
- The internal gRPC call `p.client.GetInstalledPackages(...)` stays unchanged (proto method name is not renamed)
- Search for all call sites: `\.GetInstalledPackages(` → `\.ListInstalledPackages(`

**File:** `docs/api-reference.html`

- Find entry `id: 'pkg-list'` and update `name:`, the `sig(...)` signature and example

---

### `InstallPackageFromFile` → `InstallPackage` ✅

**File:** `client/package_manager.go`

- Rename `func (p *PackageManager) InstallPackageFromFile(ctx context.Context, orbPath string) error` → `func (p *PackageManager) InstallPackage(ctx context.Context, orbPath string) error`
- Update the comment above the method
- Search for all call sites: `\.InstallPackageFromFile(` → `\.InstallPackage(`

**File:** `docs/api-reference.html`

- Find entry `id: 'pkg-install'` and update `name:`, the `sig(...)` signature and example

---

## How to apply a ⏳ pending rename

1. Apply changes in `client/<manager>.go` as described above
2. Update `docs/api-reference.html` as described above
3. Run `grep -rn "\.OldName(" .` to catch any remaining call sites
4. Mark the entry as ✅ aplicado in this file
