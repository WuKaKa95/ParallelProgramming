with Ada.Text_IO; use Ada.Text_IO;
with Ada.Numerics.Float_Random; use Ada.Numerics.Float_Random;
with Random_Seeds; use Random_Seeds;
with Ada.Real_Time; use Ada.Real_Time;
with Mutex_Bakery;
with Mutex_Dekker;
with Mutex_Peterson;

procedure Main is

begin
   Mutex_Peterson;
end Main;
