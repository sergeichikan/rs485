#include <stdio.h>
#include <unistd.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/ioctl.h>
#include <fcntl.h>
#include <termios.h>
#include <errno.h>
#include <stdlib.h>
#include <stdbool.h>

#define MANUAL_PORTDIR_CONTROL  1
//
#define PORT0_PIN_NUM           6
#define PORT1_PIN_NUM           10
//
#define PORT0_NAME              "/dev/ttyAMA2"
#define PORT1_NAME              "/dev/ttyUSB0"
//
#define NPORTS                  2
#define BAUDRATE                B19200

int configure_iopin(unsigned pin_num, const char *direction, bool writable);
int unconfigure_iopin(unsigned pin_num);
void setPinValue(int fd, bool on);
void print_buf(const char* buf, const int size);

int fd_de0, fd_de1;
int fd_port0, fd_port1;
int fd_ports[NPORTS];
int rts_flag = TIOCM_RTS;
const char* ports_name[NPORTS] = {PORT0_NAME, PORT1_NAME};
int de_pins[NPORTS];
char wbuf[64];
char rbuf[64];

int main(void) {
printf("started\n");
if (MANUAL_PORTDIR_CONTROL)
    // port 0 RE/DE pin
    fd_de0 = configure_iopin(PORT0_PIN_NUM, "out", true);
    if (fd_de0 <= 0) {
        return -1;
    }
    // port 1 RE/DE pin
    fd_de1 = configure_iopin(PORT1_PIN_NUM, "out", true);
    if (fd_de1 <= 0) {
        return -1;
    }
    // port0
    fd_port0 = open(PORT0_NAME, O_RDWR | O_NOCTTY | O_NDELAY/*O_SYNC*/);
    if (fd_port0 <= 0) {
        printf("can`t open %s\n", PORT0_NAME);
    }
    // port1
    fd_port1 = open(PORT1_NAME, O_RDWR | O_NOCTTY | O_NDELAY);
    if (fd_port1 <= 0) {
        printf("can`t open %s\n", PORT1_NAME);
    }

    fd_ports[0] = fd_port0;
    fd_ports[1] = fd_port1;
    //
    de_pins[0] = fd_de0;
    de_pins[1] = fd_de1;
    //configure ports
    for (int iport = 0; iport < NPORTS; iport++) {
        struct termios port_settings;
        int fd = fd_ports[iport];

        tcgetattr(fd, &port_settings);
        cfsetispeed(&port_settings, BAUDRATE);
        printf("port_settings c_ispeed %d\n", port_settings.c_ispeed);

        port_settings.c_cflag &= ~PARENB;          // Disables the Parity   Enable bit(PARENB),So No Parity
        printf("port_settings c_cflag %u\n", port_settings.c_cflag);
        port_settings.c_cflag &= ~CSTOPB;          // CSTOPB = 2 Stop bits,here it is cleared so 1 Stop bit
        printf("port_settings c_cflag %u\n", port_settings.c_cflag);
        port_settings.c_cflag &= ~CSIZE;           // Clears the mask for setting the data size
        printf("port_settings c_cflag %u\n", port_settings.c_cflag);
        port_settings.c_cflag |=  CS8;             // Set the data bits = 8
        printf("port_settings c_cflag %u\n", port_settings.c_cflag);

        port_settings.c_cflag &= ~CRTSCTS;         // No Hardware flow Control
        printf("port_settings c_cflag %u\n", port_settings.c_cflag);
        port_settings.c_cflag |= CREAD | CLOCAL;   // Enable receiver, ignore Modem Control lines
        printf("port_settings c_cflag %u\n", port_settings.c_cflag);

        port_settings.c_iflag &= ~(IGNBRK | BRKINT | PARMRK | ISTRIP | INLCR | IGNCR | ICRNL | IXON);
        printf("port_settings c_iflag %u\n", port_settings.c_iflag);
        port_settings.c_lflag &= ~(ECHO | ECHONL | ICANON | ISIG | IEXTEN);  // Non Cannonical mode
        printf("port_settings c_lflag %u\n", port_settings.c_lflag);
        port_settings.c_oflag &= ~OPOST;   // No Output Processing
        printf("port_settings c_oflag %u\n", port_settings.c_oflag);

        port_settings.c_cc[VMIN]  = 1;            // read doesn't block
        printf("port_settings c_cc[%d] %d\n", VMIN, port_settings.c_cc[VMIN]);
        port_settings.c_cc[VTIME] = 5*2;          // 0.5*2 seconds read timeout
        printf("port_settings c_cc[%d] %d\n", VTIME, port_settings.c_cc[VTIME]);

        if ((tcsetattr(fd, TCSANOW, &port_settings)) != 0)
            printf("can`t setting attributes for %s\n", ports_name[iport]);
    }

    unsigned ctr = 0;
    for (;;) {
    	break;
        const int sender_idx = ctr % NPORTS;
        const int receiver_idx = ++ctr % NPORTS;
        const int fd_sender = fd_ports[sender_idx];
        const int fd_receiver = fd_ports[receiver_idx];
        const int de_sender = de_pins[sender_idx];
        const int de_receiver = de_pins[receiver_idx];
        const char *sender_name = ports_name[sender_idx];
        const char *receiver_name = ports_name[receiver_idx];

        setPinValue(de_sender, true);
        setPinValue(de_receiver, false);
        usleep(100000);

        // generate random data
        for (int i = 0; i < sizeof(wbuf); i++) {
            wbuf[i] = rand() % 255;
        }

        // send data
        printf("send data to %s:\n", sender_name);
        print_buf(wbuf, sizeof(wbuf));
        int n = write(fd_sender, wbuf, sizeof(wbuf));
        tcdrain(fd_sender);
        if (n != sizeof(wbuf)) {
            printf("can`t write to %s(%d)\n", sender_name, errno);
        } else {
            printf("write %d bytes to %s ok\n", n, sender_name);
        }
        // receive data
        usleep(100000);
        n = read(fd_receiver, rbuf, sizeof(rbuf));
        if (n != sizeof(rbuf)) {
            printf("can`t read from %s(%d)\n", receiver_name, errno);
        } else {
            printf("read %d bytes from %s:\n", n, receiver_name);
            print_buf(rbuf, sizeof(rbuf));

            if (memcmp(wbuf, rbuf, sizeof(rbuf))) {
                printf("data comparing error\n");
            }
        }
        usleep(1000000);
    }

    unconfigure_iopin(PORT0_PIN_NUM);
    unconfigure_iopin(PORT1_PIN_NUM);
    close(fd_de0);
    close(fd_de1);
    close(fd_port0);
    close(fd_port1);
    return 0;
}

/* direction:
 * input = "in"
 * output = "out"
 */
int configure_iopin(unsigned pin_num, const char *direction, bool writable) {
    char pin_name[48];
    sprintf(pin_name, "%d", pin_num);
    int fd = -1;

    int export_fd = open("/sys/class/gpio/export", O_WRONLY);
    if (export_fd != -1) {
        write(export_fd, pin_name, strlen(pin_name));
        close(export_fd);

        sprintf(pin_name, "/sys/class/gpio/gpio%d/direction", pin_num);
        int dir_fd = open(pin_name, O_WRONLY);
        if (dir_fd != -1) {
            write(dir_fd, direction, strlen(direction));
            close(dir_fd);

            sprintf(pin_name, "/sys/class/gpio/gpio%d/value", pin_num);
            fd = open(pin_name, writable ? O_RDWR : O_RDONLY);
            if (fd == -1) {
                printf("pin%u:can`t open /sys/class/gpio/gpio%u/value\n", pin_num, pin_num);
            }
        } else {
            printf("pin%u:can`t set direction\n", pin_num);
        }
    } else
        printf("pin%u:can`t open /sys/class/gpio/export\n", pin_num);
    return fd;
}

int unconfigure_iopin(unsigned pin_num) {
    char pin_name[48];
    sprintf(pin_name, "%d", pin_num);

    int unexport_fd = open("/sys/class/gpio/unexport", O_WRONLY);
    if (unexport_fd != -1) {
        write(unexport_fd, pin_name, strlen(pin_name));
        close(unexport_fd);
        return -1;
    }
    return 0;
}

void setPinValue(int fd, bool on) {
    char wr_val = on ? '1' : '0';
    lseek(fd, SEEK_SET, 0);
    write(fd, &wr_val, sizeof(wr_val));
}

void print_buf(const char* buf, const int size) {
    for (int i = 0; i < size; i++) {
        printf("0x%02x ", buf[i]);
        if (((i+1) % 16) == 0) {
            printf("\n");
        }
    }
}
