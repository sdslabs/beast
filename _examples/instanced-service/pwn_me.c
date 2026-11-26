#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

// Compile with: gcc -o pwn pwn_me.c -fno-stack-protector -no-pie

void win() {
    FILE *fp;
    char flag[100];
    
    fp = fopen("/challenge/flag.txt", "r");
    if (fp == NULL) {
        printf("Error: Could not open flag file!\n");
        return;
    }
    
    if (fgets(flag, sizeof(flag), fp) != NULL) {
        printf("Congratulations! Here's your flag: %s\n", flag);
    }
    
    fclose(fp);
}

void vulnerable() {
    char buffer[64];
    
    printf("Welcome to the Instanced PWN Challenge!\n");
    printf("Each user gets their own container instance.\n");
    printf("Can you overflow the buffer and call win()?\n\n");
    printf("Enter your payload: ");
    fflush(stdout);
    
    // Vulnerable: no bounds checking!
    ssize_t n = read(0, buffer, 256);
    if (n > 0 && n < 256) {
        buffer[n] = '\0';
    }
    
    printf("You entered: %s\n", buffer);
    printf("Better luck next time!\n");
}

int main() {
    // Disable buffering for proper network I/O
    setvbuf(stdin, NULL, _IONBF, 0);
    setvbuf(stdout, NULL, _IONBF, 0);
    setvbuf(stderr, NULL, _IONBF, 0);
    
    vulnerable();
    
    return 0;
}
