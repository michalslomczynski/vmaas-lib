package vmaas

import (
	"database/sql"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/redhatinsights/vmaas-lib/vmaas/conf"
	"github.com/redhatinsights/vmaas-lib/vmaas/utils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	lock  = &sync.Mutex{}
	db    *gorm.DB
	sqlDB *sql.DB
)

var loadFuncs = []func(c *Cache){
	loadPkgNames, loadUpdates, loadUpdatesIndex, loadEvrMaps, loadArchs, loadArchCompat, loadPkgDetails,
	loadRepoDetails, loadLabel2ContentSetID, loadPkgRepos, loadErrata, loadPkgErrata, loadErrataRepoIDs,
	loadCves, loadPkgErrataModule, loadModule2IDs, loadModuleRequires, loadDBChanges, loadString,
	// OVAL
	loadOvalDefinitionDetail, loadOvalDefinitionCves, loadPackagenameID2DefinitionIDs, loadRepoCpes,
	loadContentSet2Cpes, loadCpeID2DefinitionIDs, loadOvalCriteriaDependency, loadOvalCriteriaID2Type,
	loadOvalStateID2Arches, loadOvalModuleTestDetail, loadOvalTestDetail, loadOvalTestID2States,
}

func openDB(path string) error {
	tmpDB, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return errors.Wrap(err, "couldn't open sqlite")
	}
	db = tmpDB
	sqlDB, err = db.DB()
	if err != nil {
		return errors.Wrap(err, "couldn't return *sql.DB")
	}
	return nil
}

func closeDB() {
	if err := sqlDB.Close(); err != nil {
		utils.LogWarn("err", err.Error(), "Could not close DB")
	}
	sqlDB = nil
	db = nil
}

// Make sure only one load at a time is performed
func loadCache(path string) (*Cache, error) {
	lock.Lock()
	start := time.Now()

	if err := openDB(path); err != nil {
		return nil, err
	}
	defer closeDB()

	c := Cache{}

	var wg sync.WaitGroup
	guard := make(chan struct{}, conf.Env.MaxGoroutines)
	for _, fn := range loadFuncs {
		wg.Add(1)
		guard <- struct{}{}
		go func(fn func(c *Cache)) {
			fn(&c)
			<-guard
			wg.Done()
		}(fn)
	}

	wg.Wait()
	utils.LogInfo("elapsed", time.Since(start), "Cache loaded successfully")
	lock.Unlock()
	return &c, nil
}

func loadErrataRepoIDs(c *Cache) {
	defer utils.TimeTrack(time.Now(), "ErrataID2RepoIDs")

	type ErrataRepo struct {
		ErrataID ErrataID
		RepoID   RepoID
	}
	r := ErrataRepo{}
	cnt := getCount("errata_repo", "errata_id", "errata_id,repo_id")
	m := make(map[ErrataID]map[RepoID]bool, cnt)
	rows := getAllRows("errata_repo", "errata_id,repo_id", "errata_id,repo_id")

	for rows.Next() {
		if err := rows.Scan(&r.ErrataID, &r.RepoID); err != nil {
			panic(err)
		}
		errataMap := m[r.ErrataID]
		if errataMap == nil {
			errataMap = map[RepoID]bool{}
		}
		errataMap[r.RepoID] = true
		m[r.ErrataID] = errataMap
	}
	c.ErrataID2RepoIDs = m
}

func loadPkgErrata(c *Cache) {
	cnt := getCount("pkg_errata", "pkg_id", "pkg_id,errata_id")
	pkgToErrata := make(map[PkgID][]ErrataID, cnt)
	for k, v := range loadInt2Ints("pkg_errata", "pkg_id,errata_id", "PkgID2ErrataIDs") {
		id := PkgID(k)
		for _, i := range v {
			pkgToErrata[id] = append(pkgToErrata[id], ErrataID(i))
		}
	}
	c.PkgID2ErrataIDs = pkgToErrata
}

func loadPkgRepos(c *Cache) {
	defer utils.TimeTrack(time.Now(), "PkgRepos")

	nPkg := getCount("pkg_repo", "pkg_id", "pkg_id")
	res := make(map[PkgID][]RepoID, nPkg)
	var n PkgID
	var p RepoID

	doForRows("select pkg_id, repo_id from pkg_repo", func(row *sql.Rows) {
		err := row.Scan(&n, &p)
		if err != nil {
			panic(err)
		}
		res[n] = append(res[n], p)
	})
	c.PkgID2RepoIDs = res
}

func loadPkgNames(c *Cache) {
	defer utils.TimeTrack(time.Now(), "PkgNames")

	type PkgName struct {
		ID          NameID
		Packagename string
	}

	r := PkgName{}
	cntID := getCount("packagename", "id", "id")
	cntName := getCount("packagename", "packagename", "id")
	id2name := make(map[NameID]string, cntID)
	name2id := make(map[string]NameID, cntName)
	rows := getAllRows("packagename", "id,packagename", "id")

	for rows.Next() {
		if err := rows.Scan(&r.ID, &r.Packagename); err != nil {
			panic(err)
		}
		id2name[r.ID] = r.Packagename
		name2id[r.Packagename] = r.ID
	}
	c.ID2Packagename = id2name
	c.Packagename2ID = name2id
}

func loadUpdates(c *Cache) {
	defer utils.TimeTrack(time.Now(), "Updates")

	cnt := getCount("updates", "name_id", "package_order")
	res := make(map[NameID][]PkgID, cnt)
	var n NameID
	var p PkgID
	doForRows("select name_id, package_id from updates order by package_order", func(row *sql.Rows) {
		err := row.Scan(&n, &p)
		if err != nil {
			panic(err)
		}
		res[n] = append(res[n], p)
	})
	c.Updates = res
}

func loadUpdatesIndex(c *Cache) {
	defer utils.TimeTrack(time.Now(), "Updates index")
	cnt := getCount("updates_index", "name_id", "package_order")
	res := make(map[NameID]map[EvrID][]int, cnt)
	var n NameID
	var e EvrID
	var o int
	doForRows("select name_id, evr_id, package_order from updates_index order by package_order", func(row *sql.Rows) {
		err := row.Scan(&n, &e, &o)
		if err != nil {
			panic(err)
		}
		nmap := res[n]
		if nmap == nil {
			nmap = map[EvrID][]int{}
		}
		nmap[e] = append(nmap[e], o)
		res[n] = nmap
	})
	c.UpdatesIndex = res
}

func getCount(tableName, col, orderBy string) (cnt int) {
	row := sqlDB.QueryRow(fmt.Sprintf("select count(distinct %s) from %s order by %s", col, tableName, orderBy))
	if err := row.Scan(&cnt); err != nil {
		panic(err)
	}
	return cnt
}

func getAllRows(tableName, cols, orderBy string) *sql.Rows {
	rows, err := sqlDB.Query(fmt.Sprintf("SELECT %s FROM %s ORDER BY %s",
		cols, tableName, orderBy))
	if err != nil {
		panic(err)
	}
	return rows
}

func doForRows(q string, f func(row *sql.Rows)) {
	rows, err := sqlDB.Query(q)
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		f(rows)
	}
}

func loadIntArray(tableName, col, orderBy string) []int {
	rows := getAllRows(tableName, col, orderBy)
	defer rows.Close()

	var arr []int
	var num int
	for rows.Next() {
		err := rows.Scan(&num)
		if err != nil {
			panic(err)
		}

		arr = append(arr, num)
	}
	return arr
}

func loadStrArray(tableName, col, orderBy string) []string {
	rows := getAllRows(tableName, col, orderBy)
	defer rows.Close()

	var arr []string
	var val string
	for rows.Next() {
		err := rows.Scan(&val)
		if err != nil {
			panic(err)
		}

		arr = append(arr, val)
	}
	return arr
}

func loadEvrMaps(c *Cache) {
	defer utils.TimeTrack(time.Now(), "EVR")

	type IDEvr struct {
		ID EvrID
		utils.Evr
	}

	r := IDEvr{}
	cnt := getCount("evr", "id", "id")
	id2evr := make(map[EvrID]utils.Evr, cnt)
	evr2id := map[utils.Evr]EvrID{}
	rows := getAllRows("evr", "id,epoch,version,release", "id")

	for rows.Next() {
		//nolint:typecheck,nolintlint // false-positive, r.Epoch undefined (type IDEvr has no field or method Epoch)
		if err := rows.Scan(&r.ID, &r.Epoch, &r.Version, &r.Release); err != nil {
			panic(err)
		}
		id2evr[r.ID] = r.Evr
		evr2id[r.Evr] = r.ID
	}
	c.ID2Evr = id2evr
	c.Evr2ID = evr2id
}

func loadArchs(c *Cache) {
	defer utils.TimeTrack(time.Now(), "Arch")

	type Arch struct {
		ID   ArchID
		Arch string
	}
	r := Arch{}
	cntID := getCount("arch", "id", "id")
	cntArch := getCount("arch", "arch", "id")
	id2arch := make(map[ArchID]string, cntID)
	arch2id := make(map[string]ArchID, cntArch)
	rows := getAllRows("arch", "id,arch", "id")

	for rows.Next() {
		if err := rows.Scan(&r.ID, &r.Arch); err != nil {
			panic(err)
		}
		id2arch[r.ID] = r.Arch
		arch2id[r.Arch] = r.ID
	}
	c.ID2Arch = id2arch
	c.Arch2ID = arch2id
}

func loadArchCompat(c *Cache) {
	defer utils.TimeTrack(time.Now(), "ArchCompat")

	type ArchCompat struct {
		FromArchID ArchID
		ToArchID   ArchID
	}
	r := ArchCompat{}
	cnt := getCount("arch_compat", "from_arch_id", "from_arch_id")
	m := make(map[ArchID]map[ArchID]bool, cnt)
	rows := getAllRows("arch_compat", "from_arch_id,to_arch_id", "from_arch_id,to_arch_id")

	for rows.Next() {
		if err := rows.Scan(&r.FromArchID, &r.ToArchID); err != nil {
			panic(err)
		}
		fromMap := m[r.FromArchID]
		if fromMap == nil {
			fromMap = map[ArchID]bool{}
		}
		fromMap[r.ToArchID] = true
		m[r.FromArchID] = fromMap
	}
	c.ArchCompat = m
}

func loadPkgDetails(c *Cache) {
	defer utils.TimeTrack(time.Now(), "PackageDetails, Nevra2PkgID, SrcPkgID2PkgID")

	rows := getAllRows("package_detail", "*", "ID")
	cnt := getCount("package_detail", "id", "id")
	cntSrc := getCount("package_detail", "source_package_id", "id")
	id2pkdDetail := make(map[PkgID]PackageDetail, cnt)
	nevra2id := make(map[Nevra]PkgID, cnt)
	srcPkgID2PkgID := make(map[PkgID][]PkgID, cntSrc)
	var pkgID PkgID
	for rows.Next() {
		var det PackageDetail
		err := rows.Scan(&pkgID, &det.NameID, &det.EvrID, &det.ArchID, &det.SummaryID, &det.DescriptionID,
			&det.SrcPkgID, &det.Modified)
		if err != nil {
			panic(err)
		}
		id2pkdDetail[pkgID] = det

		nevra := Nevra{det.NameID, det.EvrID, det.ArchID}
		nevra2id[nevra] = pkgID

		if det.SrcPkgID == nil {
			continue
		}

		_, ok := srcPkgID2PkgID[*det.SrcPkgID]
		if !ok {
			srcPkgID2PkgID[*det.SrcPkgID] = []PkgID{}
		}

		srcPkgID2PkgID[*det.SrcPkgID] = append(srcPkgID2PkgID[*det.SrcPkgID], pkgID)
	}
	// FIXME: build ModifiedID index (probably not needed for vulnerabilities/updates)
	c.PackageDetails = id2pkdDetail
	c.Nevra2PkgID = nevra2id
	c.SrcPkgID2PkgID = srcPkgID2PkgID
}

func loadRepoDetails(c *Cache) { //nolint: funlen
	defer utils.TimeTrack(time.Now(), "RepoIDs, RepoDetails, RepoLabel2IDs, RepoPath2IDs, ProductID2RepoIDs")

	rows := getAllRows(
		"repo_detail",
		"id,label,name,url,COALESCE(basearch,''),COALESCE(releasever,''),product,product_id,revision,third_party",
		"label",
	)
	cntRepo := getCount("repo_detail", "id", "id")
	cntLabel := getCount("repo_detail", "label", "id")
	cntURL := getCount("repo_detail", "url", "id")
	cntProd := getCount("repo_detail", "product_id", "id")
	id2repoDetail := make(map[RepoID]RepoDetail, cntRepo)
	repoLabel2id := make(map[string][]RepoID, cntLabel)
	repoPath2id := make(map[string][]RepoID, cntURL)
	prodID2RepoIDs := make(map[int][]RepoID, cntProd)
	repoIDs := []RepoID{}
	var repoID RepoID
	for rows.Next() {
		var det RepoDetail
		err := rows.Scan(&repoID, &det.Label, &det.Name, &det.URL, &det.Basearch, &det.Releasever,
			&det.Product, &det.ProductID, &det.Revision, &det.ThirdParty)
		if err != nil {
			panic(err)
		}

		repoIDs = append(repoIDs, repoID)
		id2repoDetail[repoID] = det

		_, ok := repoLabel2id[det.Label]
		if !ok {
			repoLabel2id[det.Label] = []RepoID{}
		}
		repoLabel2id[det.Label] = append(repoLabel2id[det.Label], repoID)

		if len(det.URL) > 0 {
			parsedURL, err := url.Parse(det.URL)
			if err != nil {
				utils.LogWarn("URL", det.URL, "err", err.Error(), "Malformed repository URL")
			}
			repoPath := strings.TrimSuffix(parsedURL.Path, "/")
			_, ok = repoPath2id[repoPath]
			if !ok {
				repoPath2id[repoPath] = []RepoID{}
			}
			repoPath2id[repoPath] = append(repoPath2id[repoPath], repoID)
		}

		_, ok = prodID2RepoIDs[det.ProductID]
		if !ok {
			prodID2RepoIDs[det.ProductID] = []RepoID{}
		}
		prodID2RepoIDs[det.ProductID] = append(prodID2RepoIDs[det.ProductID], repoID)
	}
	c.RepoIDs = repoIDs
	c.RepoDetails = id2repoDetail
	c.RepoLabel2IDs = repoLabel2id
	c.RepoPath2IDs = repoPath2id
	c.ProductID2RepoIDs = prodID2RepoIDs
}

func loadLabel2ContentSetID(c *Cache) {
	defer utils.TimeTrack(time.Now(), "Label2ContentSetID")

	type LabelContent struct {
		ID    ContentSetID
		Label string
	}

	r := LabelContent{}
	cnt := getCount("content_set", "id", "id")
	label2contentSetID := make(map[string]ContentSetID, cnt)
	rows := getAllRows("content_set", "id,label", "id")

	for rows.Next() {
		if err := rows.Scan(&r.ID, &r.Label); err != nil {
			panic(err)
		}
		label2contentSetID[r.Label] = r.ID
	}
	c.Label2ContentSetID = label2contentSetID
}

func loadErrata(c *Cache) {
	defer utils.TimeTrack(time.Now(), "ErrataDetail, ErrataID2Name")

	erID2cves := loadInt2Strings("errata_cve", "errata_id,cve", "erID2cves")
	erID2pkgIDs := loadInt2Ints("pkg_errata", "errata_id,pkg_id", "erID2pkgID")
	erID2modulePkgIDs := loadInt2Ints("errata_modulepkg", "errata_id,pkg_id", "erID2modulePkgIDs")
	erID2bzs := loadInt2Strings("errata_bugzilla", "errata_id,bugzilla", "erID2bzs")
	erID2refs := loadInt2Strings("errata_refs", "errata_id,ref", "erID2refs")
	erID2modules := loadErrataModules()

	cols := "ID,name,synopsis,summary,type,severity,description,solution,issued,updated,url,third_party,requires_reboot" //nolint:lll,nolintlint
	rows := getAllRows("errata_detail", cols, "ID")
	errataDetail := map[string]ErrataDetail{}
	errataID2Name := map[ErrataID]string{}
	var errataID ErrataID
	var errataName string
	for rows.Next() {
		var det ErrataDetail
		err := rows.Scan(&errataID, &errataName, &det.Synopsis, &det.Summary, &det.Type, &det.Severity,
			&det.Description, &det.Solution, &det.Issued, &det.Updated, &det.URL, &det.ThirdParty, &det.RequiresReboot)
		if err != nil {
			panic(err)
		}
		errataID2Name[errataID] = errataName

		det.ID = errataID
		if cves, ok := erID2cves[int(errataID)]; ok {
			det.CVEs = cves
		}

		if pkgIDs, ok := erID2pkgIDs[int(errataID)]; ok {
			det.PkgIDs = pkgIDs
		}

		if modulePkgIDs, ok := erID2modulePkgIDs[int(errataID)]; ok {
			det.ModulePkgIDs = modulePkgIDs
		}

		if bzs, ok := erID2bzs[int(errataID)]; ok {
			det.Bugzillas = bzs
		}

		if refs, ok := erID2refs[int(errataID)]; ok {
			det.Refs = refs
		}

		if modules, ok := erID2modules[int(errataID)]; ok {
			det.Modules = modules
		}
		errataDetail[errataName] = det
	}
	c.ErrataDetail = errataDetail
	c.ErrataID2Name = errataID2Name
}

func loadCves(c *Cache) {
	defer utils.TimeTrack(time.Now(), "CveDetail")

	cveID2cwes := loadInt2Strings("cve_cwe", "cve_id,cwe", "cveID2cwes")
	cveID2pkg := loadInt2Ints("cve_pkg", "cve_id,pkg_id", "cveID2pkg")
	cve2eid := loadString2Ints("errata_cve", "cve,errata_id", "cve2eid")

	rows := getAllRows("cve_detail", "*", "id")
	cveDetails := map[string]CveDetail{}
	cveNames := map[int]string{}
	var cveID int
	var cveName string
	for rows.Next() {
		var det CveDetail
		err := rows.Scan(&cveID, &cveName, &det.RedHatURL, &det.SecondaryURL, &det.Cvss3Score, &det.Cvss3Metrics,
			&det.Impact, &det.PublishedDate, &det.ModifiedDate, &det.Iava, &det.Description, &det.Cvss2Score,
			&det.Cvss2Metrics, &det.Source)
		if err != nil {
			panic(err)
		}

		cwes, ok := cveID2cwes[cveID]
		sort.Strings(cwes)
		if ok {
			det.CWEs = cwes
		}

		pkgs, ok := cveID2pkg[cveID]
		if ok {
			det.PkgIDs = pkgs
		}

		eids, ok := cve2eid[cveName]
		if ok {
			det.ErrataIDs = eids
		}
		cveDetails[cveName] = det
		cveNames[cveID] = cveName
	}
	c.CveDetail = cveDetails
	c.CveNames = cveNames
}

func loadPkgErrataModule(c *Cache) {
	defer utils.TimeTrack(time.Now(), "PkgErrata2Module")

	orderBy := "pkg_id,errata_id,module_stream_id"
	table := "errata_modulepkg"
	pkgIDs := loadIntArray(table, "pkg_id", orderBy)
	errataIDs := loadIntArray(table, "errata_id", orderBy)
	moduleStreamIDs := loadIntArray(table, "module_stream_id", orderBy)

	m := map[PkgErrata][]int{}

	for i := 0; i < len(pkgIDs); i++ {
		pkgErrata := PkgErrata{pkgIDs[i], errataIDs[i]}
		_, ok := m[pkgErrata]
		if !ok {
			m[pkgErrata] = []int{}
		}

		m[pkgErrata] = append(m[pkgErrata], moduleStreamIDs[i])
	}
	c.PkgErrata2Module = m
}

func loadModule2IDs(c *Cache) {
	defer utils.TimeTrack(time.Now(), "ModuleName2IDs")

	orderBy := "module,stream"
	table := "module_stream"
	modules := loadStrArray(table, "module", orderBy)
	streams := loadStrArray(table, "stream", orderBy)
	streamIDs := loadIntArray(table, "stream_id", orderBy)

	m := map[ModuleStream][]int{}

	for i := 0; i < len(modules); i++ {
		pkgErrata := ModuleStream{modules[i], streams[i]}
		_, ok := m[pkgErrata]
		if !ok {
			m[pkgErrata] = []int{}
		}

		m[pkgErrata] = append(m[pkgErrata], streamIDs[i])
	}
	c.Module2IDs = m
}

func loadModuleRequires(c *Cache) {
	defer utils.TimeTrack(time.Now(), "ModuleRequire")

	table := "module_stream_require"
	moduleRequires := loadInt2Ints(table, "stream_id,require_id", "module2requires")
	c.ModuleRequires = moduleRequires
}

func loadString(c *Cache) {
	defer utils.TimeTrack(time.Now(), "String")

	rows := getAllRows("string", "*", "ID")
	m := map[int]string{}
	var id int
	var str *string
	for rows.Next() {
		err := rows.Scan(&id, &str)
		if err != nil {
			panic(err)
		}
		if str != nil {
			m[id] = *str
		}
	}
	c.String = m
}

func loadDBChanges(c *Cache) {
	defer utils.TimeTrack(time.Now(), "DBChange")

	rows := getAllRows("dbchange", "*", "errata_changes")
	arr := []DBChange{}
	var item DBChange
	for rows.Next() {
		err := rows.Scan(&item.ErrataChanges, &item.CveChanges, &item.RepoChanges,
			&item.LastChange, &item.Exported)
		if err != nil {
			panic(err)
		}
		arr = append(arr, item)
	}
	c.DBChange = arr[0]
}

func loadInt2Ints(table, cols, info string) map[int][]int {
	defer utils.TimeTrack(time.Now(), info)

	splitted := strings.Split(cols, ",")
	cnt := getCount(table, splitted[0], cols)
	rows := getAllRows(table, cols, cols)
	int2ints := make(map[int][]int, cnt)
	var key int
	var val int
	for rows.Next() {
		err := rows.Scan(&key, &val)
		if err != nil {
			panic(err)
		}

		_, ok := int2ints[key]
		if !ok {
			int2ints[key] = []int{}
		}
		int2ints[key] = append(int2ints[key], val)
	}
	return int2ints
}

func loadInt2Strings(table, cols, info string) map[int][]string {
	defer utils.TimeTrack(time.Now(), info)

	splitted := strings.Split(cols, ",")
	cnt := getCount(table, splitted[0], cols)
	rows := getAllRows(table, cols, cols)
	int2strs := make(map[int][]string, cnt)
	var key int
	var val string
	for rows.Next() {
		err := rows.Scan(&key, &val)
		if err != nil {
			panic(err)
		}

		_, ok := int2strs[key]
		if !ok {
			int2strs[key] = []string{}
		}

		int2strs[key] = append(int2strs[key], val)
	}
	return int2strs
}

func loadString2Ints(table, cols, info string) map[string][]int {
	defer utils.TimeTrack(time.Now(), info)

	splitted := strings.Split(cols, ",")
	cnt := getCount(table, splitted[0], cols)
	rows := getAllRows(table, cols, cols)
	int2strs := make(map[string][]int, cnt)
	var key string
	var val int
	for rows.Next() {
		err := rows.Scan(&key, &val)
		if err != nil {
			panic(err)
		}

		_, ok := int2strs[key]
		if !ok {
			int2strs[key] = []int{}
		}

		int2strs[key] = append(int2strs[key], val)
	}
	return int2strs
}

func loadErrataModules() map[int][]Module {
	defer utils.TimeTrack(time.Now(), "errata2module")

	rows := getAllRows("errata_module", "*", "errata_id")

	erID2modules := map[int][]Module{}
	var erID int
	var mod Module
	for rows.Next() {
		err := rows.Scan(&erID, &mod.Name, &mod.StreamID, &mod.Stream, &mod.Version, &mod.Context)
		if err != nil {
			panic(err)
		}

		_, ok := erID2modules[erID]
		if !ok {
			erID2modules[erID] = []Module{}
		}

		erID2modules[erID] = append(erID2modules[erID], mod)
	}
	return erID2modules
}

func loadOvalDefinitionDetail(c *Cache) {
	defer utils.TimeTrack(time.Now(), "oval_definition_detail")

	type OvalDefinitionDetail struct {
		ID               DefinitionID
		DefinitionTypeID int
		CriteriaID       CriteriaID
	}

	row := OvalDefinitionDetail{}
	defDetail := make(map[DefinitionID]DefinitionDetail)
	rows := getAllRows("oval_definition_detail", "id,definition_type_id,criteria_id", "id")

	for rows.Next() {
		if err := rows.Scan(&row.ID, &row.DefinitionTypeID, &row.CriteriaID); err != nil {
			panic(err)
		}
		defDetail[row.ID] = DefinitionDetail{
			DefinitionTypeID: row.DefinitionTypeID,
			CriteriaID:       row.CriteriaID,
		}
	}
	c.OvaldefinitionDetail = defDetail
}

func loadOvalDefinitionCves(c *Cache) {
	defer utils.TimeTrack(time.Now(), "oval_definition_cve")

	type OvalDefinitionCve struct {
		DefinitionID DefinitionID
		Cve          string
	}
	r := OvalDefinitionCve{}
	ret := make(map[DefinitionID][]string)
	cols := "definition_id,cve"
	rows := getAllRows("oval_definition_cve", cols, cols)

	for rows.Next() {
		if err := rows.Scan(&r.DefinitionID, &r.Cve); err != nil {
			panic(err)
		}
		ret[r.DefinitionID] = append(ret[r.DefinitionID], r.Cve)
	}
	c.OvaldefinitionID2Cves = ret
}

func loadPackagenameID2DefinitionIDs(c *Cache) {
	defer utils.TimeTrack(time.Now(), "PackagenameID2definitionIDs")

	type NameDefinition struct {
		NameID       NameID
		DefinitionID DefinitionID
	}
	r := NameDefinition{}
	ret := make(map[NameID][]DefinitionID)
	cols := "name_id,definition_id"
	rows := getAllRows("packagename_oval_definition", cols, cols)

	for rows.Next() {
		if err := rows.Scan(&r.NameID, &r.DefinitionID); err != nil {
			panic(err)
		}
		ret[r.NameID] = append(ret[r.NameID], r.DefinitionID)
	}
	c.PackagenameID2definitionIDs = ret
}

func loadRepoCpes(c *Cache) {
	defer utils.TimeTrack(time.Now(), "RepoID2CpeIDs")

	type CpeRepo struct {
		RepoID RepoID
		CpeID  CpeID
	}
	r := CpeRepo{}
	ret := make(map[RepoID][]CpeID)
	cols := "repo_id,cpe_id"
	rows := getAllRows("cpe_repo", cols, cols)

	for rows.Next() {
		if err := rows.Scan(&r.RepoID, &r.CpeID); err != nil {
			panic(err)
		}
		ret[r.RepoID] = append(ret[r.RepoID], r.CpeID)
	}
	c.RepoID2CpeIDs = ret
}

func loadContentSet2Cpes(c *Cache) {
	defer utils.TimeTrack(time.Now(), "ContentSetID2CpeIDs")

	type CpeCS struct {
		ContentSetID ContentSetID
		CpeID        CpeID
	}
	r := CpeCS{}
	ret := make(map[ContentSetID][]CpeID)
	cols := "content_set_id,cpe_id"
	rows := getAllRows("cpe_content_set", cols, cols)

	for rows.Next() {
		if err := rows.Scan(&r.ContentSetID, &r.CpeID); err != nil {
			panic(err)
		}
		ret[r.ContentSetID] = append(ret[r.ContentSetID], r.CpeID)
	}
	c.ContentSetID2CpeIDs = ret
}

func loadCpeID2DefinitionIDs(c *Cache) {
	defer utils.TimeTrack(time.Now(), "CpeID2OvalDefinitionIDs")

	type DefinitionCpe struct {
		CpeID        CpeID
		DefinitionID DefinitionID
	}
	r := DefinitionCpe{}
	ret := make(map[CpeID][]DefinitionID)
	cols := "cpe_id,definition_id"
	rows := getAllRows("oval_definition_cpe", cols, cols)

	for rows.Next() {
		if err := rows.Scan(&r.CpeID, &r.DefinitionID); err != nil {
			panic(err)
		}
		ret[r.CpeID] = append(ret[r.CpeID], r.DefinitionID)
	}
	c.CpeID2OvalDefinitionIDs = ret
}

func loadOvalCriteriaDependency(c *Cache) {
	defer utils.TimeTrack(
		time.Now(),
		"OvalCriteriaID2DepCriteriaIDs, OvalCriteriaID2DepTestIDs, OvalCriteriaID2DepModuleTestIDs",
	)

	type OvalCriteriaDep struct {
		ParentCriteriaID CriteriaID
		DepCriteriaID    CriteriaID
		DepTestID        TestID
		DepModuleTestID  ModuleTestID
	}

	r := OvalCriteriaDep{}

	cnt := getCount("oval_criteria_dependency", "parent_criteria_id", "parent_criteria_id")
	criteriaID2DepCriteriaIDs := make(map[CriteriaID][]CriteriaID, cnt)
	criteriaID2DepTestIDs := make(map[CriteriaID][]TestID, cnt)
	criteriaID2DepModuleTestIDs := make(map[CriteriaID][]ModuleTestID, cnt)

	cols := "parent_criteria_id,COALESCE(dep_criteria_id, 0),COALESCE(dep_test_id, 0),COALESCE(dep_module_test_id, 0)"
	rows := getAllRows("oval_criteria_dependency", cols, cols)

	for rows.Next() {
		if err := rows.Scan(&r.ParentCriteriaID, &r.DepCriteriaID, &r.DepTestID, &r.DepModuleTestID); err != nil {
			panic(err)
		}
		if _, ok := criteriaID2DepCriteriaIDs[r.ParentCriteriaID]; !ok {
			criteriaID2DepCriteriaIDs[r.ParentCriteriaID] = make([]CriteriaID, 0)
			criteriaID2DepTestIDs[r.ParentCriteriaID] = make([]TestID, 0)
			criteriaID2DepModuleTestIDs[r.ParentCriteriaID] = make([]ModuleTestID, 0)
		}
		if r.DepCriteriaID != 0 {
			criteriaID2DepCriteriaIDs[r.ParentCriteriaID] = append(criteriaID2DepCriteriaIDs[r.ParentCriteriaID],
				r.DepCriteriaID)
		}
		if r.DepTestID != 0 {
			criteriaID2DepTestIDs[r.ParentCriteriaID] = append(criteriaID2DepTestIDs[r.ParentCriteriaID], r.DepTestID)
		}
		if r.DepModuleTestID != 0 {
			criteriaID2DepModuleTestIDs[r.ParentCriteriaID] = append(criteriaID2DepModuleTestIDs[r.ParentCriteriaID],
				r.DepModuleTestID)
		}
	}
	c.OvalCriteriaID2DepCriteriaIDs = criteriaID2DepCriteriaIDs
	c.OvalCriteriaID2DepTestIDs = criteriaID2DepTestIDs
	c.OvalCriteriaID2DepModuleTestIDs = criteriaID2DepModuleTestIDs
}

func loadOvalCriteriaID2Type(c *Cache) {
	defer utils.TimeTrack(time.Now(), "OvalCriteriaID2Type")

	type OvalCriteriaType struct {
		CriteriaID CriteriaID
		TypeID     int
	}

	r := OvalCriteriaType{}
	cnt := getCount("oval_criteria_type", "criteria_id", "criteria_id,type_id")
	criteriaID2Type := make(map[CriteriaID]int, cnt)
	cols := "criteria_id,type_id"
	rows := getAllRows("oval_criteria_type", cols, cols)

	for rows.Next() {
		if err := rows.Scan(&r.CriteriaID, &r.TypeID); err != nil {
			panic(err)
		}
		criteriaID2Type[r.CriteriaID] = r.TypeID
	}
	c.OvalCriteriaID2Type = criteriaID2Type
}

func loadOvalStateID2Arches(c *Cache) {
	defer utils.TimeTrack(time.Now(), "OvalModuleTestDetail")

	type StateArch struct {
		StateID OvalStateID
		ArchID  ArchID
	}
	r := StateArch{}
	ret := make(map[OvalStateID][]ArchID)
	cols := "state_id,arch_id"
	rows := getAllRows("oval_state_arch", cols, cols)

	for rows.Next() {
		if err := rows.Scan(&r.StateID, &r.ArchID); err != nil {
			panic(err)
		}
		ret[r.StateID] = append(ret[r.StateID], r.ArchID)
	}
	c.OvalStateID2Arches = ret
}

func loadOvalModuleTestDetail(c *Cache) {
	defer utils.TimeTrack(time.Now(), "OvalModuleTestDetail")

	type ModuleTestDetail struct {
		ID           ModuleTestID
		ModuleStream string
	}

	r := ModuleTestDetail{}
	details := make(map[ModuleTestID]OvalModuleTestDetail)
	cols := "id,module_stream"
	rows := getAllRows("oval_module_test_detail", cols, cols)

	for rows.Next() {
		if err := rows.Scan(&r.ID, &r.ModuleStream); err != nil {
			panic(err)
		}
		splitted := strings.Split(r.ModuleStream, ":")
		details[r.ID] = OvalModuleTestDetail{
			ModuleStream: ModuleStream{Module: splitted[0], Stream: splitted[1]},
		}
	}
	c.OvalModuleTestDetail = details
}

func loadOvalTestDetail(c *Cache) {
	defer utils.TimeTrack(time.Now(), "OvalTestDetail")

	type TestDetail struct {
		ID               TestID
		PackageNameID    NameID
		CheckExistenceID int
	}

	r := TestDetail{}
	testDetail := make(map[TestID]OvalTestDetail)
	cols := "id,package_name_id,check_existence_id"
	rows := getAllRows("oval_test_detail", cols, cols)

	for rows.Next() {
		if err := rows.Scan(&r.ID, &r.PackageNameID, &r.CheckExistenceID); err != nil {
			panic(err)
		}
		testDetail[r.ID] = OvalTestDetail{PkgNameID: r.PackageNameID, CheckExistence: r.CheckExistenceID}
	}
	c.OvalTestDetail = testDetail
}

func loadOvalTestID2States(c *Cache) {
	defer utils.TimeTrack(time.Now(), "OvalTestID2States")

	type TestState struct {
		TestID         TestID
		StateID        OvalStateID
		EvrID          EvrID
		EvrOperationID int
	}

	r := TestState{}
	test2State := make(map[TestID][]OvalState)
	cols := "test_id,state_id,evr_id,evr_operation_id"
	rows := getAllRows("oval_test_state", cols, cols)

	for rows.Next() {
		if err := rows.Scan(&r.TestID, &r.StateID, &r.EvrID, &r.EvrOperationID); err != nil {
			panic(err)
		}
		test2State[r.TestID] = append(test2State[r.TestID], OvalState{
			ID:           r.StateID,
			EvrID:        r.EvrID,
			OperationEvr: r.EvrOperationID,
		})
	}
	c.OvalTestID2States = test2State
}
