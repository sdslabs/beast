#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>

#define NUM_PWNS 6

typedef struct note_structure{
    void (*pwn_me)();
    char data[0xf0];
} note;

note *elems[NUM_PWNS] = {0};
unsigned num = 0;

void get_shell()
{
    system("/bin/sh");
}

void message()
{
  printf("Successfully deleted note.\n");
}

void menu()
{
    puts("\n--- Menu --- ");
    puts("1.New note");
    puts("2.Delete note");
    puts("3.Help");
    puts("4.Exit");
    printf("choice > ");
}

void create() {
    unsigned index;
    for (index = 0; index < NUM_PWNS; index++) {
        if (elems[index] == NULL) {
            elems[index] = malloc(sizeof(note));
            elems[index]->pwn_me = &message;
	    printf("Enter content > ");
            read(0,elems[index]->data,241);
	    printf("Created at %p !\n",elems[index]);
            break;
        }
    }

    if (index == NUM_PWNS) {
        puts("Out of memory");
    }
}
 
void delete(unsigned i) {
    if (elems[i] == NULL)
        return;
    elems[i]->pwn_me();
    free(elems[i]);
}

void help()
{
    printf("I am located at %p \n",&help);
}
int main(int argc, char **argv) {
    setbuf(stdout, NULL);
    setbuf(stdin, NULL);
    char buf[3];
    while(1)
      {
	unsigned index;
	menu();
	read(0,buf,3);
       	switch(strtol(buf,0,10))
	  {
	  case 1:
	    create();
	    break;
	  case 2:
	    printf("Enter index to delete note > ");
	    scanf("%d",&index);
	    delete(index);
	    break;
	  case 3:
	    help();
	    break;
	  case 4:
	    exit(0);
	  default:
	    printf("Nope.\n");
	  }
      }
}
	    
