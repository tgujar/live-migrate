import os
import json
import pika
import yaml

IPAddr=".".join(os.uname().nodename.split("-")[1:])
config = dict()
with open('./param.yaml', 'r') as f:
  config = yaml.load(f, Loader=yaml.FullLoader)

filename = 'data.json'
with open(filename, 'r', encoding='utf-8') as infile:
    try:
        print("Loading")
        old_data = json.load(infile)
    except:
        old_data = {}

def dict_diff(dict_a, dict_b, show_value_diff=True):
  result = {}
  result['added']   = {k: dict_b[k] for k in set(dict_b) - set(dict_a)}
  result['removed'] = {k: dict_a[k] for k in set(dict_a) - set(dict_b)}
  if show_value_diff:
    common_keys =  set(dict_a) & set(dict_b)
    result['value_diffs'] = {
      k:dict_b[k]
      for k in common_keys
      if dict_a[k] != dict_b[k]
    }
  return result


def getMetrics(metrics, stream):

    stream_metrics = stream.split('\n')
    for container in stream_metrics[1:-1]:
        metric_list = list(filter(lambda x: x!='', container.split()))
        metrics[metric_list[0]] =  {
            'name' : metric_list[1],
            'cpu_util': metric_list[2]
        }

metrics = {}
stream = os.popen('sudo podman stats --no-stream')
output = stream.read()
getMetrics(metrics, output)

for i in metrics:
    print(metrics[i])


with open(filename, 'w', encoding='utf-8') as outfile:
    json.dump(metrics, outfile)

diff = { IPAddr : dict_diff(old_data, metrics)}

connection = pika.BlockingConnection(pika.ConnectionParameters(config["rabbitmqIP"], int(config["rabbitmqPort"]), "/", pika.PlainCredentials(config["rabbimqUser"], config["rabbitmqPass"])))
channel = connection.channel()
channel.queue_declare(queue='hello')
channel.basic_publish(exchange='',
                      routing_key='hello',
                      body=json.dumps(diff))
print(" [x] Sent 'Hello World!'")
connection.close()

print(diff)