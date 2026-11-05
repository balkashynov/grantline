"""Extract Terraform mock test state and assert the real importer result.

The test_state event uses Terraform's values representation. Rewrap it in the
published state JSON envelope; do not fabricate or alter resource values.
"""
import json, subprocess, pathlib
root=pathlib.Path(__file__).resolve().parents[1]
rows=[json.loads(x) for x in (root/'integration/terraform/test-events.jsonl').read_text().splitlines()]
summary=next(r['test_summary'] for r in rows if r.get('type')=='test_summary')
assert summary['status']=='pass', summary
state=next(r['test_state'] for r in rows if r.get('type')=='test_state')
wrapped={'format_version':state['state_format_version'],'terraform_version':'1.13.5','values':{'root_module':state['root_module']}}
p=subprocess.run([str(root/'bin/grantline'),'import','-from','mongodbatlas_api_key.legacy','-to','mongodbatlas_service_account.replacement','-require-ip-list','true'],input=json.dumps(wrapped),text=True,capture_output=True)
assert p.returncode==0,p.stderr
assert 'SYNTHETIC_SECRET' not in p.stdout+p.stderr
comparison=json.loads(p.stdout)
q=subprocess.run([str(root/'bin/grantline'),'compare'],input=p.stdout,text=True,capture_output=True)
assert q.returncode==2,q.stderr
report=json.loads(q.stdout)
assert [r['role'] for r in report['removedGrants']]==['GROUP_DATA_ACCESS_READ_ONLY']
assert report['network']['removed']==[{'cidr':'192.0.2.128/25','witness':'192.0.2.128','addresses':'128','family':'IPv4'}]
assert report['addedGrants']==[] and report['network']['added']==[]
(root/'evidence/terraform-mock-comparison.json').write_text(json.dumps(comparison,indent=2)+'\n')
(root/'evidence/terraform-mock-report.json').write_text(json.dumps(report,indent=2)+'\n')
print('PASS: Terraform 1.13.5 + mongodbatlas 2.21.0 mocked apply -> state envelope -> importer -> expected diff; secret sentinel omitted.')
