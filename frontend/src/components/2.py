import random



def lat():
    a = random.randint(77,127)
    b = random.randint(77,127)
    if a==b:
        b = random.randint(77,127)
    c = b-a
    if c<0:
        c+=199
    return c

latency = 0

n_msg = 9
for i in range(1,n_msg):
    latency+=lat()
print(latency, 1+latency//199)