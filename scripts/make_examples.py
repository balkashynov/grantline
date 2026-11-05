"""Generate clearly synthetic, reproducible fixtures (no customer state)."""
from pathlib import Path
import json,copy
root=Path(__file__).resolve().parents[1]
org='111111111111111111111111';project='222222222222222222222222';key='333333333333333333333333';client='mdb_sa_id_444444444444444444444444'
def ident(label):return dict(label=label,orgId=org,grants=[dict(scope='org:'+org,role='ORG_MEMBER'),dict(scope='project:'+project,role='GROUP_READ_ONLY'),dict(scope='project:'+project,role='GROUP_DATA_ACCESS_READ_ONLY')],accessList=['192.0.2.0/24'],requireIpList=True)
base=dict(schemaVersion=1,scope='provided-snapshot',before=ident('Legacy deployment key'),after=ident('Replacement service account'))
examples={}
x=copy.deepcopy(base);x['after']['accessList']=['192.0.2.0/25','192.0.2.128/25'];examples['equivalent']=x
x=copy.deepcopy(base);x['after']['accessList']=['192.0.2.0/25'];x['after']['grants'].pop();examples['lost-access']=x
x=copy.deepcopy(base);x['before']['requireIpList']=False;x['after']['requireIpList']=False;x['after']['accessList']=[];examples['empty-list']=x
x=copy.deepcopy(base);x['before']['requireIpList']=None;x['after']['requireIpList']=None;examples['unknown-policy']=x
x=copy.deepcopy(base);x['after']['grants'][0]['role']='ORG_OWNER';examples['role-change']=x
x=copy.deepcopy(base);x['before']['accessList']=['2001:db8::/126'];x['after']['accessList']=['2001:db8::/127','2001:db8::2/128'];examples['ipv6-gap']=x
for name,data in examples.items():(root/'examples'/f'{name}.json').write_text(json.dumps(data,indent=2)+'\n')
(root/'web'/'examples.json').write_text(json.dumps(examples,indent=2)+'\n')
def res(t,n,vals):return dict(address=t+'.'+n,mode='managed',type=t,name=n,provider_name='registry.terraform.io/mongodb/mongodbatlas',schema_version=0,values=vals)
resources=[
res('mongodbatlas_api_key','legacy',dict(org_id=org,api_key_id=key,role_names=['ORG_MEMBER'],private_key='SYNTHETIC_SECRET_MUST_NEVER_APPEAR_IN_OUTPUT')),
res('mongodbatlas_service_account','replacement',dict(org_id=org,client_id=client,roles=['ORG_MEMBER'],client_secret='SYNTHETIC_SECRET_MUST_NEVER_APPEAR_IN_OUTPUT')),
res('mongodbatlas_api_key_project_assignment','legacy',dict(project_id=project,api_key_id=key,roles=['GROUP_READ_ONLY','GROUP_DATA_ACCESS_READ_ONLY'])),
res('mongodbatlas_service_account_project_assignment','replacement',dict(project_id=project,client_id=client,roles=['GROUP_READ_ONLY'])),
res('mongodbatlas_access_list_api_key','legacy',dict(org_id=org,api_key_id=key,cidr_block='192.0.2.0/24',ip_address=None)),
res('mongodbatlas_service_account_access_list_entry','replacement',dict(org_id=org,client_id=client,cidr_block='192.0.2.0/25',ip_address=None))]
state=dict(format_version='1.0',terraform_version='1.13.5',values=dict(root_module=dict(resources=resources[:2],child_modules=[dict(address='module.access',resources=[dict(r,address='module.access.'+r['address']) for r in resources[2:]])])))
(root/'examples'/'terraform-overlap.synthetic.json').write_text(json.dumps(state,indent=2)+'\n')
