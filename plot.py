import sys
import os
import matplotlib.pyplot as plt

# files batch is a dictionary {plot_name:[[log_file, legend_string, line colour],...]}
files_batch = {
    'single 10 wrk':[["GoSingle_wrk_10.log", "go", 'blueviolet'],
                        ["no_VT_GoSingle_wrk_10.log", "go, no VT", 'orangered'],
                        ["SharpSingle_wrk_10.log", "c#", 'green']],
    'single 30 wrk':[["GoSingle_wrk_30.log", "go",  'blueviolet'],
                        ["no_VT_GoSingle_wrk_30.log", "go, no VT",  'orangered'],
                        ["SharpSingle_wrk_30.log", "c#",  'green']],
    'single 100 wrk':[["GoSingle_wrk_100.log", "go",  'blueviolet'],
                        ["no_VT_GoSingle_wrk_100.log", "go, no VT",  'orangered']],
    'single 25 rate':[["GoSingle_rate_25.log", "go", 'blueviolet'],
                        ["no_VT_GoSingle_rate_25.log", "go, no VT", 'orangered'],
                        ["SharpSingle_rate_25.log", "c#", 'green']],
    'single 50 rate':[["GoSingle_rate_50.log", "go", 'blueviolet'],
                         ["no_VT_GoSingle_rate_50.log", "go, no VT", 'orangered']],
    'single 60 rate':[["GoSingle_rate_60.log", "go", 'blueviolet'],
                         ["no_VT_GoSingle_rate_60.log", "go, no VT", 'orangered']],
    'single 300 rate':[["GoSingle_rate_300.log", "go", 'blueviolet'],
                         ["no_VT_GoSingle_rate_300.log", "go, no VT", 'orangered']],
    'single 1000 rate':[["GoSingle_rate_1000.log", "go", 'blueviolet'],
                        ["no_VT_GoSingle_rate_1000.log", "go, no VT", 'orangered']],
    '4 nodes 10 wrk': [["Go4x1_wrk_10.log", "go", 'blueviolet'],
                          ["no_VT_Go4x1_wrk_10.log", "go, no VT", 'orangered'],
                          ["Sharp4x_SharpRPC_wrk_10.log", "c#", 'green'],
                          ["Sharp4x_GoRPC_wrk_10.log", "c# + go RPC", 'aqua']],
    '4 nodes 30 wrk': [["Go4x1_wrk_30.log", "go", 'blueviolet'],
                          ["no_VT_Go4x1_wrk_30.log", "go, no VT", 'orangered'],
                          ["Sharp4x_SharpRPC_wrk_30.log", "c#", 'green'],
                          ["Sharp4x_GoRPC_wrk_30.log", "c# + go RPC", 'aqua']],
    '4 nodes 100 wrk': [["Go4x1_wrk_100.log", "go", 'blueviolet'],
                          ["no_VT_Go4x1_wrk_100.log", "go, no VT", 'orangered'],
                          ["Sharp4x_GoRPC_wrk_100.log", "c# + go RPC", 'aqua']],
    '4 nodes 25 rate': [["Go4x1_rate_25.log", "go", 'blueviolet'],
                          ["no_VT_Go4x1_rate_25.log", "go, no VT", 'orangered'],
                          ["Sharp4x_SharpRPC_rate_25.log", "c#", 'green'],
                          ["Sharp4x_GoRPC_rate_25.log", "c# + go RPC", 'aqua']],
    '4 nodes 50 rate': [["Go4x1_rate_50.log", "go", 'blueviolet'],
                          ["no_VT_Go4x1_rate_50.log", "go, no VT", 'orangered'],
                          ["Sharp4x_GoRPC_rate_50.log", "c# + go RPC", 'aqua']],
    '4 nodes 60 rate': [["Go4x1_rate_60.log", "go", 'blueviolet'],
                          ["no_VT_Go4x1_rate_60.log", "go, no VT", 'orangered'],
                          ["Sharp4x_GoRPC_rate_60.log", "c# + go RPC", 'aqua']],
    '4 nodes 300 rate': [["Go4x1_rate_300.log", "go", 'blueviolet'],
                            ["no_VT_Go4x1_rate_300.log", "go, no VT", 'orangered'],
                            ["Sharp4x_GoRPC_rate_300.log", "c# + go RPC", 'aqua']],
    '4 nodes 1000 rate': [["Go4x1_rate_1000.log", "go", 'blueviolet'],
                            ["no_VT_Go4x1_rate_1000.log", "go, no VT", 'orangered'],
                            ["Sharp4x_GoRPC_rate_1000.log", "c# + go RPC", 'aqua']],

}


def plot_data(path):
    large = 22; med = 14;
    params = {'axes.titlesize': large,
              'legend.fontsize': med,
              'figure.figsize': (12, 8),
              'axes.labelsize': med,
              'axes.titlesize': med,
              'xtick.labelsize': med,
              'ytick.labelsize': med,
              'figure.titlesize': large}
    plt.rcParams.update(params)

    for name, files in files_batch.items():
        tps = [[]]*len(files)
        secondsFromStart = [[]]*len(files)
        cpu = [[]]*len(files)
        mem = [[]]*len(files)
        avgTps = []*len(files)
        tpb = [[]]*len(files)
        blockDeltaTime = [[]]*len(files)
        defaultMSPerBlock = -1

        # extract data
        for fileCounter in range(len(files)):
            file = files[fileCounter]
            secondsFromStartFile = []
            cpuFile = []
            memFile = []
            tpsFile = []
            tpbFile = []
            blockDeltaTimeFile = []
            with open(path + file[0], "r") as f:
                lines = f.readlines()
                avgTps.append(float(lines[5][6:]))
                msPerBlock = int(lines[6][20:])
                if defaultMSPerBlock == -1:
                    defaultMSPerBlock = msPerBlock
                elif defaultMSPerBlock != msPerBlock:
                    print("Error: file {} has bad DefaultMSPerBlock value. Please, check that all nodes configurations has the same MillisecondPerBlock value.".format(file[0]))
                    exit(1)
                for i in range(12, len(lines)):
                    line = lines[i]
                    cpumem = line.split('%,')
                    if len(cpumem) == 2:
                        millisecondsFromStartcpu = cpumem[0].split(', ')
                        secondsFromStartFile.append(float(millisecondsFromStartcpu[0])/1000)
                        cpuFile.append(float(millisecondsFromStartcpu[1]))
                        memFile.append(float(cpumem[1].strip(' ').strip('\n').strip('MB')))
                    else:
                        tpsStart = i + 2
                        break
                for i in range(tpsStart, len(lines)):
                    tpsFile.append(float(lines[i].split(', ')[2]))
                    tpbFile.append(int(lines[i].split(', ')[1]))
                    blockDeltaTimeFile.append(int(lines[i].split(', ')[0]))
            tps[fileCounter] = tpsFile
            cpu[fileCounter] = cpuFile
            mem[fileCounter] = memFile
            secondsFromStart[fileCounter] = secondsFromStartFile
            tpb[fileCounter] = tpbFile
            blockDeltaTime[fileCounter] = blockDeltaTimeFile

        # plot tps for `name`
        for i in range(len(files)):
            file = files[i]
            plt.plot(tps[i], label=file[1], color=file[2], linewidth=0.8)
            plt.axhline(y=avgTps[i], label=file[1] + ' avg TPS',linestyle='--', color=file[2], linewidth=0.8)
        plt.xlabel('Blocks')
        plt.ylabel('Transactions per second')
        plt.title('Transactions per second, '+name)
        plt.legend()
        plt.xlim(left=0)
        plt.ylim(bottom=0)
        plt.savefig('./img/tps_' + name.replace(' ', '_') + '.png')
        plt.close()

        # plot tpb (transactions per block) for `name`
        for i in range(len(files)):
            file = files[i]
            plt.plot(tpb[i], label=file[1], color=file[2], linewidth=0.8)
        plt.xlabel('Blocks')
        plt.ylabel('Transactions in block')
        plt.title('Transactions in block, '+name)
        plt.legend()
        plt.xlim(left=0)
        plt.ylim(bottom=0)
        plt.savefig('./img/tpb_' + name.replace(' ', '_') + '.png')
        plt.close()

        # plot milliseconds per block for `name`
        for i in range(len(files)):
            file = files[i]
            plt.plot(blockDeltaTime[i], label=file[1], color=file[2], linewidth=0.8)
        plt.axhline(y=defaultMSPerBlock, label='target value',linestyle='--', color='red', linewidth=0.8)
        plt.xlabel('Blocks')
        plt.ylabel('Milliseconds per block')
        plt.title('Milliseconds per block, '+name)
        plt.legend()
        plt.xlim(left=-1)
        plt.ylim(bottom=defaultMSPerBlock-1000)
        plt.savefig('./img/ms_per_block_' + name.replace(' ', '_') + '.png')
        plt.close()

        # plot cpu for `name`
        for i in range(len(files)):
            file = files[i]
            plt.plot(secondsFromStart[i], cpu[i], label=file[1], color=file[2], linewidth=0.8)
        plt.xlabel('Time, seconds')
        plt.ylabel('CPU, %')
        plt.title('CPU, '+name)
        plt.legend()
        plt.xlim(left=0)
        plt.ylim(bottom=0)
        plt.savefig('./img/cpu_' + name.replace(' ', '_') + '.png')
        plt.close()

        # plot memory for `name`
        for i in range(len(files)):
            file = files[i]
            plt.plot(secondsFromStart[i], mem[i], label=file[1], color=file[2], linewidth=0.8)
        plt.xlabel('Time, seconds')
        plt.ylabel('Memory, Mb')
        plt.title('Memory, '+name)
        plt.legend()
        plt.xlim(left=0)
        plt.ylim(bottom=0)
        plt.savefig('./img/mem_' + name.replace(' ', '_') + '.png')
        plt.close()


if __name__ == '__main__':
    helpMessage = 'Please, provide logs path. Example:\n\t$ python3 plot.py ./logs/'
    if len(sys.argv) < 2:
        print(helpMessage)
        exit(1)
    path = sys.argv[1]
    if not os.path.isdir(path):
        print(path+' is not a directory.')
        print(helpMessage)
        exit(1)

    if not os.path.exists('./img'):
        os.makedirs('./img')
    plot_data(path)
    print("Images successfully saved to ./img folder.")
