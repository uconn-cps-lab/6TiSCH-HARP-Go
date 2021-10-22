import json

f = open("testbed-topo.json",)
raw_topo = json.load(f)
f.close()

t = {}



# print(raw_topo)
for n in raw_topo["data"]:
    # if n["sensor_id"]!=0:
    t[n["sensor_id"]] = {"parent": n["parent"],"layer":0}

def findLayer(id):
    if id==1:
        return 0
    layer = 0
    while t[id]["parent"]!=1:
        id = t[id]["parent"]
        layer+=1
    return layer

print(t)
print(findLayer(48))

for i in t:
    t[i]["layer"]=findLayer(i)

with open('topo.json', 'w') as f:
    json.dump(t, f)