 #include <stdio.h>
 #include <stdlib.h>
 #include <linux/termios.h>

 int main(int argc, const char **argv) {
   printf("TCSETS2 = 0x%08lX\n", TCSETS2);
   printf("BOTHER  = 0x%08X\n", BOTHER);
   printf("NCCS    = %d\n",     NCCS);
   return 0;
 }

