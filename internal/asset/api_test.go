package asset

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func makePackage(t *testing.T, base,id,version,content string) string {
	t.Helper(); d:=filepath.Join(base,id+version); if err:=os.MkdirAll(filepath.Join(d,"payload"),0755);err!=nil{t.Fatal(err)}
	m:=Manifest{SchemaVersion:1,ID:id,Type:TypeSkill,Version:version,Paths:Paths{Payload:"payload"}};b,_:=json.Marshal(m);_ = os.WriteFile(filepath.Join(d,"asset.json"),b,0644);_ = os.WriteFile(filepath.Join(d,"payload","SKILL.md"),[]byte(content),0644)
	out:=filepath.Join(base,id+version+".aicoding.zip");mgr:=NewManager(base,DefaultAdapters()...);if _,err:=mgr.Pack(context.Background(),d,out);err!=nil{t.Fatal(err)};return out
}

func TestLifecycleAndRollback(t *testing.T){root:=t.TempDir();_ = os.MkdirAll(filepath.Join(root,".aicoding"),0755);p1:=makePackage(t,root,"demo","1.0.0","one");m:=NewManager(root,DefaultAdapters()...);if _,err:=m.Install(context.Background(),p1,ModeManaged);err!=nil{t.Fatal(err)};if err:=m.Verify(context.Background(),"demo");err!=nil{t.Fatal(err)};p2:=makePackage(t,root,"demo","2.0.0","two");if _,err:=m.Update(context.Background(),p2);err!=nil{t.Fatal(err)};if _,err:=m.Rollback("demo");err!=nil{t.Fatal(err)};b,err:=os.ReadFile(filepath.Join(root,".aicoding","assets","demo","SKILL.md"));if err!=nil||string(b)!="one"{t.Fatalf("rollback failed: %q %v",b,err)};if _,err=m.Uninstall(context.Background(),"demo",false);err!=nil{t.Fatal(err)}}
func TestMergeAndConfigSet(t *testing.T){m:=MergeConfig(ConfigLayers{Defaults:map[string]any{"a":map[string]any{"b":1.0}},User:map[string]any{"a":map[string]any{"c":2.0}}});a:=m["a"].(map[string]any);if a["b"]!=1.0||a["c"]!=2.0{t.Fatal(m)};p:=filepath.Join(t.TempDir(),"x.json");if err:=SetConfig(p,"rules.encoding","GBK");err!=nil{t.Fatal(err)};v,_:=LoadJSONMap(p);if v["rules"].(map[string]any)["encoding"]!="GBK"{t.Fatal(v)}}
func TestRejectTraversal(t *testing.T){m:=Manifest{SchemaVersion:1,ID:"x",Type:TypeSkill,Version:"1",Paths:Paths{Payload:"../x"}};if ValidateManifest(m)==nil{t.Fatal("expected traversal rejection")}}
